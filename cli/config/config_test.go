package config

import (
	"os"
	"path/filepath"
	"testing"
)

// inDir runs the test from dir, restoring the working directory afterwards,
// because Load discovers .env relative to the working directory.
func inDir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestLoadPinsTheAPIVersion(t *testing.T) {
	t.Setenv("LEXSELECT_API_KEY", "")
	t.Setenv("LEXSELECT_API_URL", "")
	inDir(t, t.TempDir())

	cfg := Load()
	if cfg.APIVersion != "2026-10-20" {
		t.Errorf("APIVersion = %q, want 2026-10-20", cfg.APIVersion)
	}
	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, DefaultAPIURL)
	}
}

func TestLoadFindsDotEnvInAParentDirectory(t *testing.T) {
	t.Setenv("LEXSELECT_API_KEY", "")
	t.Setenv("LEXSELECT_API_URL", "")
	root := t.TempDir()
	env := "# comment\nLEXSELECT_API_KEY=lxs_from_file\nLEXSELECT_API_URL=https://dev.example/api/\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "cli")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	inDir(t, sub)

	cfg := Load()
	if cfg.APIKey != "lxs_from_file" {
		t.Errorf("APIKey = %q, want the .env value", cfg.APIKey)
	}
	if cfg.APIURL != "https://dev.example/api" {
		t.Errorf("APIURL = %q, want the .env value without its trailing slash", cfg.APIURL)
	}
}

func TestLoadPrefersTheEnvironmentOverDotEnv(t *testing.T) {
	t.Setenv("LEXSELECT_API_KEY", "lxs_from_env")
	t.Setenv("LEXSELECT_API_URL", "https://env.example/api")
	root := t.TempDir()
	env := "LEXSELECT_API_KEY=lxs_from_file\nLEXSELECT_API_URL=https://file.example/api\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	inDir(t, root)

	cfg := Load()
	if cfg.APIKey != "lxs_from_env" || cfg.APIURL != "https://env.example/api" {
		t.Errorf("got key %q url %q, want the environment's values", cfg.APIKey, cfg.APIURL)
	}
}
