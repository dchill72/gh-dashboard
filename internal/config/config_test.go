package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoadFromFile_MissingFile(t *testing.T) {
	_, err := loadFromFile(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadFromFile_InvalidTOML(t *testing.T) {
	path := writeConfig(t, "this is not [ valid toml !!!!")
	_, err := loadFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}

func TestLoadFromFile_DefaultHost(t *testing.T) {
	path := writeConfig(t, `
[[orgs]]
name = "my-org"
`)
	cfg, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Host != "github.com" {
		t.Errorf("expected default host %q, got %q", "github.com", cfg.GitHub.Host)
	}
}

func TestLoadFromFile_ExplicitHost(t *testing.T) {
	path := writeConfig(t, `
[github]
host = "github.example.com"

[[orgs]]
name = "my-org"
`)
	cfg, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Host != "github.example.com" {
		t.Errorf("expected host %q, got %q", "github.example.com", cfg.GitHub.Host)
	}
}

func TestLoadFromFile_MultipleOrgs(t *testing.T) {
	path := writeConfig(t, `
[github]
host = "github.com"

[[orgs]]
name = "org-a"

[[orgs]]
name = "org-b"
repos = ["repo1", "repo2"]
`)
	cfg, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Orgs) != 2 {
		t.Fatalf("expected 2 orgs, got %d", len(cfg.Orgs))
	}
	if cfg.Orgs[0].Name != "org-a" {
		t.Errorf("expected org name %q, got %q", "org-a", cfg.Orgs[0].Name)
	}
	if len(cfg.Orgs[0].Repos) != 0 {
		t.Errorf("expected no repos for org-a, got %v", cfg.Orgs[0].Repos)
	}
	if cfg.Orgs[1].Name != "org-b" {
		t.Errorf("expected org name %q, got %q", "org-b", cfg.Orgs[1].Name)
	}
	if len(cfg.Orgs[1].Repos) != 2 {
		t.Errorf("expected 2 repos for org-b, got %v", cfg.Orgs[1].Repos)
	}
}
