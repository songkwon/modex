package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileConfig is the top-level structure for an optional YAML application config file.
// Use this for settings that describe application behavior/semantics rather than
// per-deployment infrastructure wiring (the latter belongs in environment variables).
type FileConfig struct {
	Auth AuthSection `yaml:"auth"`
}

type AuthSection struct {
	// UserMapping controls which OIDC/JWT claims are mapped to user identity fields.
	// This is a classic example of application-level configuration that is nicer
	// in a versioned config file than scattered environment variables.
	UserMapping UserMapping `yaml:"user_mapping"`
}

// UserMapping defines claim names coming from the identity provider (Keycloak etc.).
// These are used during OIDC login to populate the local User record.
type UserMapping struct {
	// UniqueIDClaim is the claim whose *value* is treated as the user's stable unique identifier.
	// The company convention is to use email as the cross-system unique key.
	UniqueIDClaim string `yaml:"unique_id_claim"`

	// AvatarClaim holds the user's profile picture URL.
	// Common values: "picture", "avatar", or a custom mapper like "wxPhotoURL".
	AvatarClaim string `yaml:"avatar_claim"`

	// DisplayNameClaim is the primary human name shown in the UI (the "1级显示的名字").
	DisplayNameClaim string `yaml:"display_name_claim"`

	// SecondaryInfoClaim is the value shown below the primary name
	// (e.g. department, organization, title). It is currently surfaced as the
	// "department" field for backward compatibility with existing UI and admin pages.
	SecondaryInfoClaim string `yaml:"secondary_info_claim"`
}

// Load reads the application config.
// Precedence (lowest to highest):
//  1. Sensible defaults inside the code that calls this.
//  2. Values from the YAML config file (if found).
//  3. Explicit environment variable overrides (highest — allows per-deployment tweaks
//     without modifying the committed config file).
//
// The config file location is determined by:
//   - CONFIG_FILE environment variable (highest priority for location)
//   - Then a short list of conventional locations (./config.yaml, ./configs/config.yaml, /etc/modex/config.yaml)
func Load() (FileConfig, error) {
	path, err := findConfigPath()
	if err != nil {
		return FileConfig{}, err
	}
	if path == "" {
		return FileConfig{}, nil // no config file configured or present — this is fine
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, err
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, err
	}

	log.Printf("loaded application config from %s", path)
	return fc, nil
}

// LoadUserMapping returns the effective UserMapping after applying file + env precedence.
func LoadUserMapping() UserMapping {
	fc, err := Load()
	if err != nil {
		log.Printf("warning: failed to load config file for user mapping, falling back to env only: %v", err)
	}

	m := fc.Auth.UserMapping

	// Environment variables take highest priority (explicit overrides).
	// This preserves the previous pure-env behavior and allows emergency / per-env changes.
	if v := os.Getenv("OIDC_CLAIM_UNIQUE_ID"); v != "" {
		m.UniqueIDClaim = v
	}
	if v := os.Getenv("OIDC_CLAIM_AVATAR"); v != "" {
		m.AvatarClaim = v
	}
	if v := os.Getenv("OIDC_CLAIM_DISPLAY_NAME"); v != "" {
		m.DisplayNameClaim = v
	}
	if v := os.Getenv("OIDC_CLAIM_SECONDARY_INFO"); v != "" {
		m.SecondaryInfoClaim = v
	}

	return m
}

// findConfigPath returns the first existing config file path according to the lookup rules.
// It does not return an error if nothing is found (the caller decides what to do).
func findConfigPath() (string, error) {
	candidates := []string{}

	if explicit := os.Getenv("CONFIG_FILE"); explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("%w: %s", ErrConfigNotFound, explicit)
			}
			return "", err
		}
		return explicit, nil
	}

	// Conventional locations.
	// Order matters: more specific / container-friendly paths first where it makes sense.
	candidates = append(candidates,
		// Common container locations (matches our Dockerfile WORKDIR and common k8s mounts)
		"/app/config.yaml",
		"/app/config.yml",
		"/app/configs/config.yaml",
		"/app/configs/config.yml",

		// Relative to current working directory (good for local dev and simple deploys)
		"config.yaml",
		"config.yml",
		filepath.Join("configs", "config.yaml"),
		filepath.Join("configs", "config.yml"),

		// System-wide locations
		"/etc/modex/config.yaml",
		"/etc/modex/config.yml",
	)

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil
}

// MustLoad is like Load but logs and returns an empty config on error (never panics the server).
func MustLoad() FileConfig {
	fc, err := Load()
	if err != nil {
		log.Printf("config load error (continuing with empty config): %v", err)
		return FileConfig{}
	}
	return fc
}

// ErrConfigNotFound is returned when an explicit CONFIG_FILE was given but the file does not exist.
var ErrConfigNotFound = errors.New("config file not found")
