package journal

import (
	"strconv"
	"strings"
	"time"
)

// Stamp is what the journal needs to know about a commit beyond its row:
// when it was committed (local time) and whether it is the working copy.
type Stamp struct {
	Time        time.Time
	WorkingCopy bool
}

// TimestampTemplate keys each commit by the same shortest commit id the log
// marker carries (jj.LogTemplate), so rows and stamps join on Commit.CommitId.
// Fields: id, local committer time, "@" for the working copy.
const TimestampTemplate = `commit_id.shortest() ++ "\t" ++ committer.timestamp().local().format("%Y-%m-%dT%H:%M:%S%z") ++ "\t" ++ if(current_working_copy, "@", "-") ++ "\n"`

const timestampLayout = "2006-01-02T15:04:05-0700"

// TimestampArgs is the `jj log` invocation producing TimestampTemplate output.
// It runs after the row query, which already snapshotted the working copy,
// so --ignore-working-copy is safe and faster.
func TimestampArgs(revset string, limit int) []string {
	args := []string{"log", "--no-graph", "--color", "never", "--quiet", "--ignore-working-copy",
		"-r", revset, "-T", TimestampTemplate}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args
}

// ParseTimestamps parses TimestampTemplate output. Malformed lines are
// skipped and counted so a partial failure never hides rows.
func ParseTimestamps(output []byte) (map[string]Stamp, int) {
	stamps := make(map[string]Stamp)
	skipped := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			skipped++
			continue
		}
		when, err := time.Parse(timestampLayout, strings.TrimSpace(fields[1]))
		if err != nil {
			skipped++
			continue
		}
		stamp := Stamp{Time: when}
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) == "@" {
			stamp.WorkingCopy = true
		}
		stamps[strings.TrimSpace(fields[0])] = stamp
	}
	return stamps, skipped
}
