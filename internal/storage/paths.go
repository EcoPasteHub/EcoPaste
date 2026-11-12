package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultBundleID = "com.ayangweb.eco-paste"
	manifestName    = "storage.json"
	identityName    = ".ecopaste-storage.json"
)

// Paths is the resolved persistent data layout.
type Paths struct {
	Root      string
	DBDir     string
	DB        string
	Resources string
	Config    string
	State     string
}

type manifest struct {
	Version     int       `json:"version"`
	Environment string    `json:"environment"`
	DataDir     string    `json:"dataDir"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ResolvePaths returns the production data root unless ECOPASTE_DATA_DIR
// overrides it. The override keeps tests and development data isolated.
func ResolvePaths() (Paths, error) {
	if root := os.Getenv("ECOPASTE_DATA_DIR"); root != "" {
		return layout(root), nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user config directory: %w", err)
	}

	bootstrap := filepath.Join(base, defaultBundleID, "prod")
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		return Paths{}, fmt.Errorf("create bootstrap directory: %w", err)
	}

	defaultRoot := bootstrap
	root, err := readManifest(bootstrap, defaultRoot)
	if err != nil {
		// The Rust implementation falls back to the default root and rewrites
		// the manifest. Preserve that recovery behavior without hiding errors.
		root = defaultRoot
		_ = writeManifest(bootstrap, root)
	}

	return layout(root), nil
}

func layout(root string) Paths {
	return Paths{
		Root:      root,
		DBDir:     filepath.Join(root, "db"),
		DB:        filepath.Join(root, "db", "clipboard.db"),
		Resources: filepath.Join(root, "resources"),
		Config:    filepath.Join(root, "config"),
		State:     filepath.Join(root, "state"),
	}
}

func readManifest(bootstrap, defaultRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(bootstrap, manifestName))
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(defaultRoot, 0o755); err != nil {
			return "", err
		}
		if err := writeManifest(bootstrap, defaultRoot); err != nil {
			return "", err
		}
		return defaultRoot, nil
	}
	if err != nil {
		return "", err
	}

	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	if m.Version != 1 || m.Environment != "prod" || m.DataDir == "" {
		return "", fmt.Errorf("storage manifest does not match the production environment")
	}
	if m.DataDir == defaultRoot {
		return m.DataDir, nil
	}
	if err := validateIdentity(m.DataDir); err != nil {
		return "", err
	}
	return m.DataDir, nil
}

func validateIdentity(root string) error {
	raw, err := os.ReadFile(filepath.Join(root, identityName))
	if err != nil {
		return err
	}
	var identity struct {
		Version     int    `json:"version"`
		Environment string `json:"environment"`
	}
	if err := json.Unmarshal(raw, &identity); err != nil {
		return err
	}
	if identity.Version != 1 || identity.Environment != "prod" {
		return fmt.Errorf("storage identity does not match the production environment")
	}
	return nil
}

func writeManifest(bootstrap, root string) error {
	raw, err := json.MarshalIndent(manifest{
		Version:     1,
		Environment: "prod",
		DataDir:     root,
		UpdatedAt:   time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(bootstrap, manifestName), raw, 0o644)
}
