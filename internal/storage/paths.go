package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

const (
	manifestName = "storage.json"
	identityName = ".ecopaste-storage.json"

	manifestVersion = 1

	Dev  = "dev"
	Prod = "prod"

	// DevelopmentBundleID separates local builds from the published app.
	DevelopmentBundleID = "com.ayangweb.eco-paste.dev"
)

// Paths are the directories EcoPaste persists to. The bootstrap directory
// always holds the manifest that points at the real data root, so a
// custom location can be changed without losing the anchor.
type Paths struct {
	Base      string
	Bootstrap string
	Root      string
	DB        string
	Resources string
	Config    string
	State     string
	Logs      string
}

type manifest struct {
	Version     int       `json:"version"`
	Environment string    `json:"environment"`
	DataDir     string    `json:"dataDir"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type identity struct {
	Version     int       `json:"version"`
	Environment string    `json:"environment"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Mode reports the data environment of this build: dev builds keep their
// data apart from a release install.
func Mode() string {
	bundled := false
	if exe, err := os.Executable(); err == nil {
		bundled = strings.Contains(exe, ".app/Contents/MacOS/")
	}
	return resolveMode(core.App().ID, isDevBuild, os.Getenv("ECOPASTE_MODE"), bundled)
}

func resolveMode(bundleID string, devBuild bool, override string, bundled bool) string {
	if modeForIdentity(bundleID, devBuild) == Dev || !bundled {
		return Dev
	}
	if override == Dev || override == Prod {
		return override
	}
	return Prod
}

func modeForIdentity(bundleID string, devBuild bool) string {
	if bundleID == DevelopmentBundleID || devBuild {
		return Dev
	}
	return Prod
}

// DefaultBase is the application support directory that holds every
// environment of this bundle.
func DefaultBase() (string, error) {
	if base := os.Getenv("ECOPASTE_HOME"); base != "" {
		return filepath.Abs(base)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve application support: %w", err)
	}
	return filepath.Join(dir, "com.ayangweb.eco-paste"), nil
}

// Default resolves the paths of the current build.
func Default() (*Paths, error) {
	base, err := DefaultBase()
	if err != nil {
		return nil, err
	}
	return Resolve(base, Mode())
}

// Resolve reads the bootstrap manifest under base/environment and
// returns the data root it names, falling back to the default root when
// the manifest, target directory or identity is unusable.
func Resolve(base, environment string) (*Paths, error) {
	if environment != Dev && environment != Prod {
		return nil, fmt.Errorf("unknown environment %q", environment)
	}
	bootstrap := filepath.Join(base, environment)
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		return nil, fmt.Errorf("create bootstrap dir: %w", err)
	}

	root := bootstrap
	m, err := readManifest(filepath.Join(bootstrap, manifestName))
	switch {
	case err == nil && m.Version == manifestVersion && m.Environment == environment && validRoot(m.DataDir, bootstrap, environment):
		root = m.DataDir
	case err != nil && !errors.Is(err, os.ErrNotExist):
		root = bootstrap
	}

	if root == bootstrap {
		if err := writeManifest(bootstrap, manifest{Version: manifestVersion, Environment: environment, DataDir: bootstrap, UpdatedAt: time.Now().UTC()}); err != nil {
			return nil, err
		}
	}

	paths := &Paths{
		Base:      base,
		Bootstrap: bootstrap,
		Root:      root,
		DB:        filepath.Join(root, "db"),
		Resources: filepath.Join(root, "resources"),
		Config:    filepath.Join(root, "config"),
		State:     filepath.Join(root, "state"),
		Logs:      filepath.Join(root, "logs"),
	}
	for _, dir := range []string{paths.Root, paths.DB, paths.Resources, paths.Config, paths.State, paths.Logs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return paths, nil
}

// ValidateTarget reports whether dir can hold EcoPaste data: an empty
// directory, or one whose identity manifest matches this environment.
func ValidateTarget(dir, environment string) error {
	if environment != Dev && environment != Prod {
		return fmt.Errorf("unknown environment %q", environment)
	}
	data, err := os.ReadFile(filepath.Join(dir, identityName))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read storage identity: %w", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("read storage target: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("target directory is not an EcoPaste data directory")
		}
		return nil
	}

	var id identity
	if err := json.Unmarshal(data, &id); err != nil {
		return fmt.Errorf("parse storage identity: %w", err)
	}
	if id.Version != manifestVersion || id.Environment != environment {
		return fmt.Errorf("target directory belongs to another EcoPaste environment")
	}
	return nil
}

// SetRoot points the bootstrap manifest at dir after validating it.
func SetRoot(base, environment, dir string) error {
	if err := ValidateTarget(dir, environment); err != nil {
		return err
	}
	return CommitRoot(base, environment, dir)
}

// CommitRoot writes dir's identity and points the bootstrap manifest at
// it. The caller must have validated dir with ValidateTarget before
// moving data into it.
func CommitRoot(base, environment, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	id := identity{Version: manifestVersion, Environment: environment, CreatedAt: time.Now().UTC()}
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return fmt.Errorf("encode storage identity: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, identityName), data, 0o644); err != nil {
		return fmt.Errorf("write storage identity: %w", err)
	}
	bootstrap := filepath.Join(base, environment)
	if err := os.MkdirAll(bootstrap, 0o755); err != nil {
		return fmt.Errorf("create bootstrap dir: %w", err)
	}
	return writeManifest(bootstrap, manifest{Version: manifestVersion, Environment: environment, DataDir: dir, UpdatedAt: time.Now().UTC()})
}

func validRoot(dir, fallback, environment string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	if dir == fallback {
		return true
	}
	return ValidateTarget(dir, environment) == nil
}

func readManifest(path string) (manifest, error) {
	var m manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

func writeManifest(bootstrap string, m manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode storage manifest: %w", err)
	}
	path := filepath.Join(bootstrap, manifestName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write storage manifest: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace storage manifest: %w", err)
	}
	return nil
}
