package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentityVarsSelectConfigPaths(t *testing.T) {
	origApp, origEnv, origRepo := AppDirName, EnvConfigDirName, RepoConfigDirName
	t.Cleanup(func() { AppDirName, EnvConfigDirName, RepoConfigDirName = origApp, origEnv, origRepo })
	AppDirName, EnvConfigDirName, RepoConfigDirName = "jjj", "JJJ_CONFIG_DIR", ".jjj"

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("JJUI_CONFIG_DIR", t.TempDir()) // jjui's variable must be ignored
	t.Setenv("JJJ_CONFIG_DIR", "")

	want := filepath.Join(home, ".config", "jjj", "config.toml")
	if got := getConfigFilePath(); got != want {
		t.Fatalf("global config path: got %q want %q", got, want)
	}

	envDir := t.TempDir()
	t.Setenv("JJJ_CONFIG_DIR", envDir)
	if got, want := getConfigFilePath(), filepath.Join(envDir, "config.toml"); got != want {
		t.Fatalf("env config path: got %q want %q", got, want)
	}

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".jjj"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".jjj", "config.toml"), []byte("limit = 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := LoadRepoConfigFile(repo)
	if err != nil {
		t.Fatalf("LoadRepoConfigFile: %v", err)
	}
	if string(data) != "limit = 7\n" {
		t.Fatalf("repo config: got %q", data)
	}
}
