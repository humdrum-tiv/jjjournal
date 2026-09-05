// Package statusinfo gathers and formats what the Status panel shows. Pure
// functions except Gather, which runs read-only jj queries.
package statusinfo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appContext "github.com/idursun/jjui/internal/ui/context"
)

type Info struct {
	Repo      string
	ChangeID  string
	Bookmark  string // nearest bookmark at or below @; "" when none
	Remote    string
	Ahead     int
	Behind    int
	HasRemote bool // Ahead/Behind are meaningful (the remote bookmark exists)
	FetchedAt time.Time
	Revset    string
	Mode      string // "journal" or "graph"
}

var readOnly = []string{"--no-graph", "--color", "never", "--quiet", "--ignore-working-copy"}

func WorkingCopyArgs() []string {
	return append([]string{"log", "-r", "@", "-T", `change_id.shortest(8) ++ "\n"`}, readOnly...)
}

// ClosestBookmarkArgs lists the local bookmarks on the nearest bookmarked
// ancestor of @ (including @ itself).
func ClosestBookmarkArgs() []string {
	return append([]string{"log", "-r", "heads(::@ & bookmarks())", "-T", `local_bookmarks.map(|b| b.name()).join(",") ++ "\n"`}, readOnly...)
}

// CountArgs prints one "x" per commit in revset; Count sums them.
func CountArgs(revset string) []string {
	return append([]string{"log", "-r", revset, "-T", `"x"`}, readOnly...)
}

func FirstLine(output []byte) string {
	line, _, _ := strings.Cut(string(output), "\n")
	return strings.TrimSpace(line)
}

func FirstBookmark(output []byte) string {
	name, _, _ := strings.Cut(FirstLine(output), ",")
	return strings.TrimSpace(name)
}

func Count(output []byte) int { return bytes.Count(output, []byte("x")) }

// FetchTime is the modification time of git's FETCH_HEAD (colocated repos
// first, then jj's internal git store); zero when the repo was never fetched.
func FetchTime(repoRoot string) time.Time {
	for _, p := range []string{
		filepath.Join(repoRoot, ".git", "FETCH_HEAD"),
		filepath.Join(repoRoot, ".jj", "repo", "store", "git", "FETCH_HEAD"),
	} {
		if st, err := os.Stat(p); err == nil {
			return st.ModTime()
		}
	}
	return time.Time{}
}

func FormatAge(now, at time.Time) string {
	if at.IsZero() {
		return "never"
	}
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Lines renders the three Status panel lines.
func (i Info) Lines(now time.Time) [3]string {
	fetched := "fetched " + FormatAge(now, i.FetchedAt)
	var second string
	switch {
	case i.Bookmark == "":
		second = "(no bookmark) · " + fetched
	case i.HasRemote:
		second = fmt.Sprintf("%s ↑%d ↓%d %s · %s", i.Bookmark, i.Ahead, i.Behind, i.Remote, fetched)
	default:
		second = fmt.Sprintf("%s · not on %s · %s", i.Bookmark, i.Remote, fetched)
	}
	return [3]string{
		fmt.Sprintf("%s  @ %s", i.Repo, i.ChangeID),
		second,
		fmt.Sprintf("%s · %s", i.Mode, i.Revset),
	}
}

// Gather runs the read-only queries. Any failure leaves that field at its
// zero value; the panel must never block or error the UI.
func Gather(runner appContext.CommandRunner, repoRoot, remote, revset, mode string, now time.Time) Info {
	info := Info{Repo: filepath.Base(repoRoot), Remote: remote, Revset: revset, Mode: mode, FetchedAt: FetchTime(repoRoot)}
	if out, err := runner.RunCommandImmediate(WorkingCopyArgs()); err == nil {
		info.ChangeID = FirstLine(out)
	}
	if out, err := runner.RunCommandImmediate(ClosestBookmarkArgs()); err == nil {
		info.Bookmark = FirstBookmark(out)
	}
	if info.Bookmark != "" && remote != "" {
		local, remoteRef := info.Bookmark, info.Bookmark+"@"+remote
		aheadOut, aheadErr := runner.RunCommandImmediate(CountArgs(remoteRef + ".." + local))
		behindOut, behindErr := runner.RunCommandImmediate(CountArgs(local + ".." + remoteRef))
		if aheadErr == nil && behindErr == nil {
			info.HasRemote = true
			info.Ahead, info.Behind = Count(aheadOut), Count(behindOut)
		}
	}
	return info
}
