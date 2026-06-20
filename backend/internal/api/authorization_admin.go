package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/store"
)

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	if user, ok := s.app.Auth().CurrentUser(r); ok {
		fresh, err := s.app.Store().UserByID(user.ID)
		return fresh, err == nil
	}
	if tok := bearerToken(r); tok != "" {
		if user, _, _, err := s.app.Store().UserByOAuthAccessToken(tok); err == nil {
			return user, true
		}
	}
	return store.User{}, false
}

func isAdmin(u store.User) bool {
	for _, r := range u.Roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// canManageCategory reports whether the user may manage the given platform.
// A managed category id covers its descendants (e.g. "engineering" covers
// "engineering.cbb").
func canManageCategory(u store.User, categoryID string) bool {
	for _, m := range u.ManagedCategories {
		if m == categoryID || strings.HasPrefix(categoryID, m+".") {
			return true
		}
	}
	return false
}

func canManageCategories(u store.User, categoryIDs []string) bool {
	for _, id := range categoryIDs {
		if canManageCategory(u, id) {
			return true
		}
	}
	return false
}

// isTeamLeader checks if the user is the designated leader of the team.
func (s *Server) isTeamLeader(u store.User, teamKey string) bool {
	if teamKey == "" {
		return false
	}
	t, err := s.app.Store().Team(teamKey)
	if err != nil {
		return false
	}
	for _, l := range t.Leaders {
		if strings.EqualFold(l, u.Username) || strings.EqualFold(l, u.ID) {
			return true
		}
	}
	return false
}

// teamMembers returns usernames/ids in the team (for ownership checks).
func (s *Server) teamMembers(teamKey string) []string {
	return s.app.Store().TeamMembers(teamKey)
}

// canManageViaResponsibleTeam allows members (incl. leader) of a category's responsible team
// to manage that category's resources (generic domain ownership).
func (s *Server) canManageViaResponsibleTeam(u store.User, categoryIDs []string) bool {
	for _, cid := range categoryIDs {
		// Check direct; for hierarchy the responsible on parent covers subs conceptually,
		// but we also check the specific id's assignment.
		resp := s.categoryResponsible(cid)
		if resp == "" {
			continue
		}
		for _, m := range s.teamMembers(resp) {
			if strings.EqualFold(m, u.Username) || strings.EqualFold(m, u.ID) {
				return true
			}
		}
	}
	return false
}

func (s *Server) categoryResponsible(id string) string {
	// Walk the tree (small data) to find assignment for id or nearest ancestor with one.
	tree := s.app.Store().CategoryTree()
	var find func([]store.Category) string
	find = func(cats []store.Category) string {
		for _, c := range cats {
			if c.ID == id {
				if c.ResponsibleTeam != "" {
					return c.ResponsibleTeam
				}
				// inherit from parent? caller walks up if needed; here return what we have at leaf.
				return ""
			}
			if hit := find(c.Children); hit != "" {
				return hit
			}
		}
		return ""
	}
	// Also try ancestor match for sub-ids (e.g. "standards.foo" covered by "standards" team)
	for _, c := range tree {
		if c.ID == id || strings.HasPrefix(id, c.ID+".") {
			if c.ResponsibleTeam != "" {
				return c.ResponsibleTeam
			}
		}
		for _, ch := range c.Children {
			if ch.ID == id || strings.HasPrefix(id, ch.ID+".") {
				if ch.ResponsibleTeam != "" {
					return ch.ResponsibleTeam
				}
			}
		}
	}
	return find(tree)
}

// accessibleCategoryIDs returns the set of category IDs a user may see/manage in
// the admin console. Super admins get (nil, true) meaning "everything". A team
// member gets the categories their team(s) own (Category.ResponsibleTeam) plus
// all descendants. A user with no team gets an empty set.
func (s *Server) accessibleCategoryIDs(u store.User) (set map[string]bool, all bool) {
	if s.app.Auth().IsSuperAdmin(u) {
		return nil, true
	}
	teamKeys := map[string]bool{}
	for _, k := range s.app.Store().TeamKeysForUser(u) {
		teamKeys[strings.ToLower(strings.TrimSpace(k))] = true
	}
	set = map[string]bool{}
	if len(teamKeys) == 0 {
		return set, false
	}
	cats := s.app.Store().AllCategories()
	for _, c := range cats {
		if c.ResponsibleTeam != "" && teamKeys[strings.ToLower(c.ResponsibleTeam)] {
			set[c.ID] = true
		}
	}
	// Expand to all descendants via ParentID closure (a child without its own
	// ResponsibleTeam is still owned through its parent).
	for {
		added := false
		for _, c := range cats {
			if !set[c.ID] && c.ParentID != "" && set[c.ParentID] {
				set[c.ID] = true
				added = true
			}
		}
		if !added {
			break
		}
	}
	return set, false
}

// hasConsoleAccess reports whether the user may enter the admin console at all
// (super admin, or a member/leader of any team).
func (s *Server) hasConsoleAccess(u store.User) bool {
	return s.app.Auth().IsSuperAdmin(u) || len(s.app.Store().TeamKeysForUser(u)) > 0
}

// isTeamAdmin reports a non-super-admin who belongs to at least one team (the
// team-scoped admin tier).
func (s *Server) isTeamAdmin(u store.User) bool {
	return !s.app.Auth().IsSuperAdmin(u) && len(s.app.Store().TeamKeysForUser(u)) > 0
}

// requireConsole gates console-scoped reads (logs, releases, module list). Super
// admins and team members pass; everyone else gets 403.
func (s *Server) requireConsole(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.app.Auth().CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return store.User{}, false
	}
	if !s.hasConsoleAccess(user) {
		writeError(w, http.StatusForbidden, "forbidden", "console access required")
		return store.User{}, false
	}
	return user, true
}

// docCategoryIDs resolves a document id to the categories of its module.
func (s *Server) docCategoryIDs(docID string) []string {
	if strings.TrimSpace(docID) == "" {
		return nil
	}
	p, err := s.app.Store().Page(docID)
	if err != nil {
		return nil
	}
	return s.moduleCategories(p.ModuleKey)
}

// mcpLogCategoryIDs best-effort resolves an MCP log to categories by parsing its
// free-form input JSON for a doc id or module key. MCP logs carry no structured
// module field, so this is heuristic; unresolved logs return nil (hidden from
// team admins, shown to super admins who bypass scoping).
func (s *Server) mcpLogCategoryIDs(inputJSON string) []string {
	if strings.TrimSpace(inputJSON) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &m); err != nil {
		return nil
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	if doc := str("doc_id", "docID", "docId"); doc != "" {
		if cats := s.docCategoryIDs(doc); len(cats) > 0 {
			return cats
		}
	}
	if mod := str("module_key", "moduleKey", "module"); mod != "" {
		return s.moduleCategories(mod)
	}
	return nil
}

// categoriesIntersect reports whether any of the ids is in the accessible set.
func categoriesIntersect(ids []string, set map[string]bool) bool {
	for _, id := range ids {
		if set[id] {
			return true
		}
	}
	return false
}

// requireUser writes 401 and returns false when no valid session is present.
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.app.Auth().CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return store.User{}, false
	}
	return user, true
}

// requireSuperAdmin gates super-admin-only actions (e.g. user/permission mgmt).
func (s *Server) requireSuperAdmin(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if !s.app.Auth().IsSuperAdmin(user) {
		writeError(w, http.StatusForbidden, "forbidden", "super admin required")
		return store.User{}, false
	}
	return user, true
}

// requirePlatform gates platform-scoped writes: super admins pass; otherwise the
// user must have management rights on at least one of the target categories.
// Team responsible for the domain (via Category.ResponsibleTeam) also grants access
// to its members/leaders (generic for OSS doc maintenance teams owning 领域).
func (s *Server) requirePlatform(w http.ResponseWriter, r *http.Request, categoryIDs []string) (store.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if s.app.Auth().IsSuperAdmin(user) {
		return user, true
	}
	if isAdmin(user) && canManageCategories(user, categoryIDs) {
		return user, true
	}
	if s.canManageViaResponsibleTeam(user, categoryIDs) {
		return user, true
	}
	writeError(w, http.StatusForbidden, "forbidden", "no management permission for this platform")
	return store.User{}, false
}

// moduleCategories resolves the category IDs attached to a module key.
func (s *Server) moduleCategories(moduleKey string) []string {
	if m, err := s.app.Store().Module(moduleKey); err == nil {
		return m.CategoryIDs
	}
	return nil
}

func (s *Server) handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusOK, s.app.Store().CategoryTree())
		return
	}
	var c store.Category
	if err := decodeBody(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	// Top-level platforms are super-admin only; sub-platforms can be created by
	// a manager of the parent platform.
	if c.ParentID == "" {
		if _, ok := s.requireSuperAdmin(w, r); !ok {
			return
		}
	} else if _, ok := s.requirePlatform(w, r, []string{c.ParentID}); !ok {
		return
	}
	created, err := s.app.Store().CreateCategory(c)
	s.writeMutation(w, created, http.StatusCreated, err)
}

func (s *Server) handleAdminCategoryByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/categories/")
	// Drag-and-drop move: POST /api/admin/categories/{id}/move {parent_id, index}.
	if strings.HasSuffix(id, "/move") {
		id = strings.TrimSuffix(id, "/move")
		if _, ok := s.requireSuperAdmin(w, r); !ok {
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
			return
		}
		var body struct {
			ParentID string `json:"parent_id"`
			Index    int    `json:"index"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		moved, err := s.app.Store().MoveCategory(id, body.ParentID, body.Index)
		s.writeMutation(w, moved, http.StatusOK, err)
		return
	}
	if _, ok := s.requirePlatform(w, r, []string{id}); !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		var c store.Category
		if err := decodeBody(r, &c); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.app.Store().UpdateCategory(id, c)
		s.writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.app.Store().DeleteCategory(id); err != nil {
			writeResult(w, nil, err)
			return
		}
		s.writeMutation(w, map[string]any{"status": "deleted", "id": id}, http.StatusOK, nil)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT or DELETE")
	}
}

func (s *Server) handleAdminModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		user, ok := s.requireConsole(w, r)
		if !ok {
			return
		}
		modules := s.app.Store().Modules("", "")
		// Team admins only see modules attached to a category they own.
		if set, all := s.accessibleCategoryIDs(user); !all {
			scoped := modules[:0:0]
			for _, m := range modules {
				if categoriesIntersect(m.CategoryIDs, set) {
					scoped = append(scoped, m)
				}
			}
			modules = scoped
		}
		if kw := keywordOf(r); kw != "" {
			filtered := modules[:0:0]
			for _, m := range modules {
				if containsFold(m.Name, kw) || containsFold(m.ModuleKey, kw) || containsFold(m.RepoURL, kw) || containsFold(m.Description, kw) {
					filtered = append(filtered, m)
				}
			}
			modules = filtered
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(modules, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, modules)
		return
	}
	var m store.Module
	if err := decodeBody(r, &m); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	// Every doc source must be filed under at least one category.
	if len(m.CategoryIDs) == 0 {
		writeError(w, http.StatusBadRequest, "category_required", "文档源必须关联至少一个分类")
		return
	}
	if _, ok := s.requirePlatform(w, r, m.CategoryIDs); !ok {
		return
	}
	created, err := s.app.Store().CreateModule(m)
	s.writeMutation(w, created, http.StatusCreated, err)
}

func (s *Server) handleAdminModuleRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/modules/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "admin module route not found")
		return
	}
	moduleKey := parts[0]
	// Migration (reassign platform/owner) has its own dual-platform permission
	// check, so handle it before the generic platform gate.
	if len(parts) == 2 && parts[1] == "migrate" && r.Method == http.MethodPost {
		s.handleMigrateModule(w, r, moduleKey)
		return
	}
	user, ok := s.requirePlatform(w, r, s.moduleCategories(moduleKey))
	if !ok {
		return
	}
	// Reveal / rotate the CI deploy token (kept out of normal serialization).
	if len(parts) == 2 && parts[1] == "deploy-token" {
		m, err := s.app.Store().Module(moduleKey)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "module not found")
			return
		}
		if r.Method == http.MethodPost { // rotate
			token := "mdx_" + strconv.FormatInt(time.Now().UnixNano(), 36)
			if _, err := s.app.Store().UpdateModule(moduleKey, store.Module{DeployToken: token}); err != nil {
				writeResult(w, nil, err)
				return
			}
			m.DeployToken = token
		} else if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deploy_token": m.DeployToken,
			"deploy_url":   firstNonEmptyStr(os.Getenv("APP_BASE_URL"), "") + "/api/deploy",
			"module_key":   moduleKey,
		})
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodPut:
		var m store.Module
		if err := decodeBody(r, &m); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		// A doc source must keep at least one category; reject an explicit clear.
		if m.CategoryIDs != nil && len(m.CategoryIDs) == 0 {
			writeError(w, http.StatusBadRequest, "category_required", "文档源必须关联至少一个分类")
			return
		}
		// Team admins may only file modules under categories they own.
		if len(m.CategoryIDs) > 0 {
			if set, all := s.accessibleCategoryIDs(user); !all {
				for _, cid := range m.CategoryIDs {
					if !set[cid] {
						writeError(w, http.StatusForbidden, "forbidden", "只能选择本团队负责的分类")
						return
					}
				}
			}
		}
		updated, err := s.app.Store().UpdateModule(moduleKey, m)
		s.writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 2 && parts[1] == "versions" && r.Method == http.MethodPost:
		var v store.Version
		if err := decodeBody(r, &v); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.app.Store().CreateVersion(moduleKey, v)
		s.writeMutation(w, created, http.StatusCreated, err)
	case len(parts) == 3 && parts[1] == "versions" && r.Method == http.MethodPut:
		var v store.Version
		if err := decodeBody(r, &v); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.app.Store().UpdateVersion(moduleKey, parts[2], v)
		s.writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 4 && parts[1] == "versions" && parts[3] == "entries" && r.Method == http.MethodPost:
		var e store.Entry
		if err := decodeBody(r, &e); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.app.Store().CreateEntry(moduleKey, parts[2], e)
		s.writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusNotFound, "not_found", "admin module route not found")
	}
}

// handleMigrateModule reassigns a module to different platform(s) (and
// optionally a new owner). The caller must be able to manage both the source
// and destination platforms (super admins bypass the check).
func (s *Server) handleMigrateModule(w http.ResponseWriter, r *http.Request, moduleKey string) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req struct {
		CategoryIDs []string `json:"category_ids"`
		OwnerGroup  string   `json:"owner_group"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.CategoryIDs) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "category_ids is required")
		return
	}
	if !s.app.Auth().IsSuperAdmin(user) {
		source := s.moduleCategories(moduleKey)
		if !isAdmin(user) || !canManageCategories(user, source) || !canManageCategories(user, req.CategoryIDs) {
			writeError(w, http.StatusForbidden, "forbidden", "need management permission on both source and target platforms")
			return
		}
	}
	names := make([]string, 0, len(req.CategoryIDs))
	for _, id := range req.CategoryIDs {
		names = append(names, s.app.Store().CategoryName(id))
	}
	updated, err := s.app.Store().UpdateModule(moduleKey, store.Module{
		CategoryIDs:  req.CategoryIDs,
		CategoryPath: strings.Join(names, " / "),
		OwnerGroup:   req.OwnerGroup,
	})
	s.writeMutation(w, updated, http.StatusOK, err)
}

func (s *Server) handleAdminEntryByID(w http.ResponseWriter, r *http.Request) {
	entryID := strings.TrimPrefix(r.URL.Path, "/api/admin/entries/")
	if moduleKey, ok := s.app.Store().EntryModuleKey(entryID); ok {
		if _, ok := s.requirePlatform(w, r, s.moduleCategories(moduleKey)); !ok {
			return
		}
	} else if _, ok := s.requireUser(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		var e store.Entry
		if err := decodeBody(r, &e); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.app.Store().UpdateEntry(entryID, e)
		s.writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.app.Store().DeleteEntry(entryID); err != nil {
			writeResult(w, nil, err)
			return
		}
		s.writeMutation(w, map[string]any{"status": "deleted", "id": entryID}, http.StatusOK, nil)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT or DELETE")
	}
}

func (s *Server) handleReleaseRoutes(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/releases/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "release route not found")
		return
	}
	releaseID := parts[0]
	rel, err := s.app.Store().Release(releaseID)
	if err != nil {
		writeResult(w, rel, err)
		return
	}
	if set, all := s.accessibleCategoryIDs(user); !all && !categoriesIntersect(s.moduleCategories(rel.ModuleKey), set) {
		writeError(w, http.StatusForbidden, "forbidden", "no access to this release")
		return
	}
	if len(parts) == 2 && parts[1] == "rollback" && r.Method == http.MethodPost {
		rel, err = s.app.Store().RollbackRelease(releaseID)
		s.writeMutation(w, rel, http.StatusOK, err)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, rel)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST /rollback")
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		users := s.app.Store().Users(r.URL.Query().Get("keyword"))
		// Enrich with the effective super admin status (persisted flag OR env SUPER_ADMIN_USERS).
		for i := range users {
			users[i].SuperAdmin = s.app.Auth().IsSuperAdmin(users[i])
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(users, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var u store.User
		if err := decodeBody(r, &u); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.app.Store().CreateUser(u)
		if err == nil {
			created.SuperAdmin = s.app.Auth().IsSuperAdmin(created)
		}
		s.writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminUserByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	switch r.Method {
	case http.MethodGet:
		u, err := s.app.Store().UserByID(id)
		if err == nil {
			u.SuperAdmin = s.app.Auth().IsSuperAdmin(u)
		}
		writeResult(w, u, err)
	case http.MethodPut:
		var u store.User
		if err := decodeBody(r, &u); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.app.Store().UpdateUser(id, u)
		if err == nil {
			updated.SuperAdmin = s.app.Auth().IsSuperAdmin(updated)
		}
		s.writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.app.Store().DeleteUser(id); err != nil {
			writeResult(w, nil, err)
			return
		}
		s.writeMutation(w, map[string]any{"status": "deleted", "id": id}, http.StatusOK, nil)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET, PUT or DELETE")
	}
}

func (s *Server) handleAdminTeams(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		teams := s.app.Store().Teams()
		if kw := keywordOf(r); kw != "" {
			filtered := teams[:0:0]
			for _, t := range teams {
				if containsFold(t.Name, kw) || containsFold(t.Key, kw) || containsFold(strings.Join(t.Leaders, " "), kw) || containsFold(t.Description, kw) || containsFold(strings.Join(t.Members, " "), kw) {
					filtered = append(filtered, t)
				}
			}
			teams = filtered
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(teams, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, teams)
	case http.MethodPost:
		var t store.Team
		if err := decodeBody(r, &t); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.app.Store().CreateTeam(t)
		s.writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminTeamRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/teams/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "team route not found")
		return
	}
	key := parts[0]
	switch {
	case len(parts) == 1:
		// View: super or any member of the team. Mutations: leader or super.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		tm, err := s.app.Store().Team(key)
		if err != nil {
			writeResult(w, tm, err)
			return
		}
		isMember := false
		for _, m := range tm.Members {
			if strings.EqualFold(m, user.Username) || strings.EqualFold(m, user.ID) {
				isMember = true
				break
			}
		}
		if !s.app.Auth().IsSuperAdmin(user) && !isMember {
			writeError(w, http.StatusForbidden, "forbidden", "team membership or super required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, tm)
		case http.MethodPut:
			if !s.app.Auth().IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
				writeError(w, http.StatusForbidden, "forbidden", "only leader or super can update team")
				return
			}
			var t store.Team
			if err := decodeBody(r, &t); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			updated, err := s.app.Store().UpdateTeam(key, t)
			s.writeMutation(w, updated, http.StatusOK, err)
		case http.MethodDelete:
			if _, ok := s.requireSuperAdmin(w, r); !ok {
				return
			}
			if err := s.app.Store().DeleteTeam(key); err != nil {
				writeResult(w, nil, err)
				return
			}
			s.writeMutation(w, map[string]any{"status": "deleted", "key": key}, http.StatusOK, nil)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET/PUT/DELETE")
		}
	case len(parts) == 2 && parts[1] == "members" && r.Method == http.MethodPost:
		// Leader or super can pull (add) members.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		if !s.app.Auth().IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
			writeError(w, http.StatusForbidden, "forbidden", "only team leader or super can add members")
			return
		}
		var req struct {
			Username string `json:"username"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if strings.TrimSpace(req.Username) == "" {
			writeError(w, http.StatusBadRequest, "invalid_input", "username required")
			return
		}
		updated, err := s.app.Store().AddTeamMember(key, req.Username)
		s.writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 3 && parts[1] == "members" && r.Method == http.MethodDelete:
		// Leader or super can remove member.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		if !s.app.Auth().IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
			writeError(w, http.StatusForbidden, "forbidden", "only team leader or super can remove members")
			return
		}
		member := parts[2]
		updated, err := s.app.Store().RemoveTeamMember(key, member)
		s.writeMutation(w, updated, http.StatusOK, err)
	default:
		writeError(w, http.StatusNotFound, "not_found", "team route not found")
	}
}
