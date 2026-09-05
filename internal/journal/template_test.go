package journal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newJJRepo creates a throwaway colocated-free jj repo with one described commit.
func newJJRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not on PATH")
	}
	dir := t.TempDir()
	cfg := filepath.Join(dir, "jj-config.toml")
	if err := os.WriteFile(cfg, []byte("[user]\nname = \"t\"\nemail = \"t@example.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "JJ_CONFIG="+cfg)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("jj %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run("git", "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("commit", "-m", "feat: first")
	return dir
}

func TestRowTemplatesRenderOneLine(t *testing.T) {
	dir := newJJRepo(t)
	for name, tmpl := range map[string]string{"graph": RowTemplateGraph, "journal": RowTemplateJournal} {
		cmd := exec.Command("jj", "log", "-r", "@-", "--no-graph", "--color", "never", "-T", tmpl)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "JJ_CONFIG="+filepath.Join(dir, "jj-config.toml"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s template failed: %v\n%s", name, err, out)
		}
		line := strings.TrimRight(string(out), "\n")
		if strings.Contains(line, "\n") {
			t.Fatalf("%s template rendered more than one line: %q", name, line)
		}
		if !strings.Contains(line, "feat: first") {
			t.Fatalf("%s template lost the subject: %q", name, line)
		}
	}
}

func TestJournalTemplateShowsWorkingCopyMarker(t *testing.T) {
	dir := newJJRepo(t)
	cmd := exec.Command("jj", "log", "-r", "@", "--no-graph", "--color", "never", "-T", RowTemplateJournal)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "JJ_CONFIG="+filepath.Join(dir, "jj-config.toml"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "(working copy)") {
		t.Fatalf("empty undescribed @ should read (working copy): %q", out)
	}
}
