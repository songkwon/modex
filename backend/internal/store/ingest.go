package store

import (
	"strconv"
	"strings"
	"time"
)

func (s *MemoryStore) IngestArtifact(a DeployArtifact) (DeployResult, error) {
	if strings.TrimSpace(a.ModuleKey) == "" || strings.TrimSpace(a.DocsVersion) == "" || len(a.Entries) == 0 || len(a.Documents) == 0 {
		return DeployResult{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	moduleName := firstNonEmpty(a.ModuleName, a.ModuleKey)
	moduleIdx, err := s.moduleIndexLocked(a.ModuleKey)
	if err != nil {
		s.modules = append(s.modules, Module{
			ID:               s.nextIDLocked("m"),
			ModuleKey:        a.ModuleKey,
			Name:             moduleName,
			Description:      a.Description,
			OwnerGroup:       firstNonEmpty(firstString(a.Authors), "docs"),
			RepoType:         firstNonEmpty(a.RepoType, "git"),
			RepoURL:          a.RepoURL,
			SourceType:       "gitlab",
			GitLabBranch:     a.Branch,
			DefaultVersion:   a.DocsVersion,
			Visibility:       "internal",
			Status:           "active",
			PackageName:      a.ModuleKey,
			PackageVersion:   a.PackageVersion,
			Channel:          "docs",
			Edition:          a.Edition,
			Keywords:         cloneStrings(a.Keywords),
			Maintainers:      cloneStrings(a.Authors),
			LastSyncedCommit: a.CommitSHA,
			LastSyncedAt:     now,
			UpdatedAt:        now,
		})
		moduleIdx = len(s.modules) - 1
	} else {
		m := &s.modules[moduleIdx]
		m.Name = moduleName
		if a.Description != "" {
			m.Description = a.Description
		}
		m.DefaultVersion = a.DocsVersion
		if a.PackageVersion != "" {
			m.PackageVersion = a.PackageVersion
		}
		if a.Edition != "" {
			m.Edition = a.Edition
		}
		if len(a.Keywords) > 0 {
			m.Keywords = cloneStrings(a.Keywords)
		}
		if len(a.Authors) > 0 {
			m.Maintainers = cloneStrings(a.Authors)
			if m.OwnerGroup == "" {
				m.OwnerGroup = a.Authors[0]
			}
		}
		if m.Status == "" {
			m.Status = "active"
		}
		if m.Visibility == "" {
			m.Visibility = "internal"
		}
		// Refresh source metadata from each CI push (repo/branch/commit).
		if a.RepoURL != "" {
			m.RepoURL = a.RepoURL
		}
		if a.RepoType != "" {
			m.RepoType = a.RepoType
		}
		if a.Branch != "" {
			m.GitLabBranch = a.Branch
		}
		if a.CommitSHA != "" {
			m.LastSyncedCommit = a.CommitSHA
		}
		m.LastSyncedAt = now
		m.UpdatedAt = now
	}
	// Preserve existing category assignment and rebuild the display path from
	// current categories so the admin UI and module cards show the right labels.
	module := &s.modules[moduleIdx]
	module.CategoryPath = s.categoryPathLocked(module.CategoryIDs)
	versionFound := false
	for i := range s.versions {
		if strings.EqualFold(s.versions[i].ModuleKey, a.ModuleKey) && s.versions[i].DocsVersion == a.DocsVersion {
			v := &s.versions[i]
			v.DisplayName = firstNonEmpty(v.DisplayName, a.DocsVersion)
			v.IsDefault = true
			v.Status = "active"
			v.PackageVersion = a.PackageVersion
			v.Edition = a.Edition
			if v.VersionType == "" {
				v.VersionType = "release"
			}
			if v.SupportStatus == "" {
				v.SupportStatus = "supported"
			}
			versionFound = true
		} else if strings.EqualFold(s.versions[i].ModuleKey, a.ModuleKey) {
			s.versions[i].IsDefault = false
		}
	}
	if !versionFound {
		s.versions = append(s.versions, Version{
			ID:             s.nextIDLocked("v"),
			ModuleKey:      a.ModuleKey,
			DocsVersion:    a.DocsVersion,
			DisplayName:    a.DocsVersion,
			VersionType:    "release",
			IsDefault:      true,
			Status:         "active",
			PackageVersion: a.PackageVersion,
			Channel:        firstNonEmpty(module.Channel, "docs"),
			Edition:        a.Edition,
			SupportStatus:  "supported",
			CreatedAt:      now,
		})
	}
	s.entries = removeEntries(s.entries, a.ModuleKey, a.DocsVersion)
	for i, e := range a.Entries {
		s.entries = append(s.entries, Entry{
			ID:          s.nextIDLocked("e"),
			ModuleKey:   a.ModuleKey,
			DocsVersion: a.DocsVersion,
			EntryKey:    e.Key,
			Title:       e.Title,
			EntryType:   firstNonEmpty(e.Type, "markdown"),
			Builder:     firstNonEmpty(e.Type, "markdown"),
			Source:      e.Source,
			StorageURI:  "memory://" + routeKey(a.ModuleKey, a.DocsVersion, e.Key),
			NavURI:      "memory://" + routeKey(a.ModuleKey, a.DocsVersion, ""),
			IndexStatus: "indexed",
			IsPrimary:   i == 0,
			SortOrder:   i + 1,
			Status:      "active",
			CreatedAt:   now,
		})
	}
	s.pages = removePages(s.pages, a.ModuleKey, a.DocsVersion)
	// Drop cached embeddings for this module/version so re-published content is
	// re-embedded on the next reindex (or lazily during search).
	if s.embeddings != nil {
		embPrefix := a.ModuleKey + ":" + a.DocsVersion + ":"
		for docID := range s.embeddings {
			if strings.HasPrefix(docID, embPrefix) {
				delete(s.embeddings, docID)
			}
		}
	}
	for _, d := range a.Documents {
		entryKey := firstNonEmpty(d.EntryKey, entryKeyFromDocID(d.DocID))
		docID := firstNonEmpty(d.DocID, a.ModuleKey+":"+a.DocsVersion+":"+entryKey)
		s.pages = append(s.pages, Page{
			ID:             s.nextIDLocked("p"),
			DocID:          docID,
			ModuleKey:      a.ModuleKey,
			ModuleName:     moduleName,
			DocsVersion:    a.DocsVersion,
			PackageVersion: firstNonEmpty(d.PackageVersion, a.PackageVersion),
			EntryKey:       entryKey,
			EntryType:      firstNonEmpty(d.EntryType, entryTypeForEntry(a.Entries, entryKey)),
			Title:          firstNonEmpty(d.Title, titleForEntry(a.Entries, entryKey)),
			Description:    d.Description,
			Path:           docPagePath(a.ModuleKey, a.DocsVersion, entryKey, firstNonEmpty(d.EntryType, entryTypeForEntry(a.Entries, entryKey)), d.Path),
			SourceFile:     d.SourceFile,
			DocType:        firstNonEmpty(d.EntryType, entryTypeForEntry(a.Entries, entryKey)),
			Status:         firstNonEmpty(d.Status, "active"),
			OwnerGroup:     module.OwnerGroup,
			CategoryIDs:    cloneStrings(module.CategoryIDs),
			Tags:           cloneStrings(coalesceStrings(d.Keywords, a.Keywords)),
			ContentText:    d.Content,
			ContentMD:      d.ContentMD,
			UpdatedAt:      now,
		})
	}
	if s.navs == nil {
		s.navs = map[string][]NavItem{}
	}
	s.navs[routeKey(a.ModuleKey, a.DocsVersion, "")] = cloneNav(a.Nav)
	if s.html == nil {
		s.html = map[string]string{}
	}
	if s.siteFiles == nil {
		s.siteFiles = map[string]SiteFile{}
	}
	for k := range s.html {
		prefix := routeKey(a.ModuleKey, a.DocsVersion, "") + ":"
		if strings.HasPrefix(k, prefix) {
			delete(s.html, k)
		}
	}
	for k := range s.siteFiles {
		prefix := routeKey(a.ModuleKey, a.DocsVersion, "") + ":"
		if strings.HasPrefix(k, prefix) {
			delete(s.siteFiles, k)
		}
	}
	for _, e := range a.Entries {
		html := htmlForEntry(a.SiteHTML, e.Key)
		if html != "" {
			s.html[routeKey(a.ModuleKey, a.DocsVersion, e.Key)] = html
		}
	}
	for name, content := range a.SiteFiles {
		entryKey, relName, ok := splitSiteFile(name)
		if !ok {
			continue
		}
		s.siteFiles[siteFileKey(a.ModuleKey, a.DocsVersion, entryKey, relName)] = SiteFile{
			Name: relName, Content: append([]byte(nil), content...), ContentType: contentTypeForName(relName, content),
		}
	}
	rel := Release{
		ID:              s.nextIDLocked("r"),
		ReleaseID:       "rel-" + strings.ToLower(a.ModuleKey) + "-" + strings.ToLower(a.DocsVersion) + "-" + strconv.FormatInt(now.UnixNano(), 36),
		ModuleKey:       a.ModuleKey,
		DocsVersion:     a.DocsVersion,
		Publisher:       firstNonEmpty(firstString(a.Authors), "docsctl"),
		BuildSystem:     "docsctl",
		TriggerType:     firstNonEmpty(a.TriggerType, "manual"),
		SourceIP:        a.SourceIP,
		ArtifactVersion: now.Format("20060102.150405"),
		PackageVersion:  a.PackageVersion,
		StorageURI:      "memory://" + a.ModuleKey + "/" + a.DocsVersion + "/docs-artifact.zip",
		Status:          "published",
		PublishedAt:     now,
		CreatedAt:       now,
	}
	s.releases = append(s.releases, rel)
	return DeployResult{Release: rel, PagesIndexed: len(a.Documents), EntriesIndexed: len(a.Entries), HTMLFiles: len(a.SiteHTML), SiteFiles: len(a.SiteFiles), BytesReceived: a.Bytes}, nil
}
