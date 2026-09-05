package journal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimestamps(t *testing.T) {
	out := []byte("0123abcd\t2026-09-04T16:30:00-0700\t@\n" +
		"deadbeef\t2026-07-13T08:55:03-0700\t-\n" +
		"garbage line without a tab\n" +
		"cafe\tnot-a-time\t-\n")
	stamps, skipped := ParseTimestamps(out)

	require.Equal(t, 2, skipped)
	require.Len(t, stamps, 2)
	require.True(t, stamps["0123abcd"].WorkingCopy)
	require.False(t, stamps["deadbeef"].WorkingCopy)
	want := time.Date(2026, 7, 13, 8, 55, 3, 0, time.FixedZone("", -7*3600))
	require.True(t, stamps["deadbeef"].Time.Equal(want), "got %v", stamps["deadbeef"].Time)
}

func TestTimestampArgs(t *testing.T) {
	args := TimestampArgs("all()", 5000)
	require.Equal(t, []string{"log", "--no-graph", "--color", "never", "--quiet", "--ignore-working-copy",
		"-r", "all()", "-T", TimestampTemplate, "--limit", "5000"}, args)
	require.NotContains(t, TimestampArgs("all()", 0), "--limit")
}
