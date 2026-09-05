package journal

import (
	"bytes"
	"io"
	"strconv"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

// LogArgs is `jj log --no-graph` for revset with jjui's row marker and
// rowTemplate. Unlike TimestampArgs it snapshots the working copy, exactly
// as the graph loader does.
func LogArgs(revset string, limit int, rowTemplate string) []string {
	args := []string{"log", "--no-graph", "--color", "always", "--quiet", "-r", revset, "-T", jj.LogTemplate(rowTemplate)}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args
}

// Source is a revisions.RowSource that serves the whole revset in one batch,
// sorted newest-first with day and week header lines (see Arrange).
type Source struct {
	batch         parser.RowBatch
	headerIndices []int
	served        bool
}

// NewSource runs the row query and the stamp query and arranges the rows.
// It returns io.EOF for an empty revset, matching graph.NewGraphStreamer,
// so the caller's "No revisions found" message keeps working.
func NewSource(runner appContext.CommandRunner, revset, rowTemplate string, limit int, opts Options) (*Source, error) {
	out, err := runner.RunCommandImmediate(LogArgs(revset, limit, rowTemplate))
	if err != nil {
		return nil, err
	}
	rows := parser.ParseRows(bytes.NewReader(out))
	if len(rows) == 0 {
		return nil, io.EOF
	}
	stampOut, err := runner.RunCommandImmediate(TimestampArgs(revset, limit))
	if err != nil {
		return nil, err
	}
	stamps, _ := ParseTimestamps(stampOut)
	arranged := Arrange(rows, stamps, opts)
	return &Source{
		batch:         parser.RowBatch{Rows: arranged.Rows, HasMore: false},
		headerIndices: arranged.HeaderIndices,
	}, nil
}

// RequestMore hands out the arranged rows once; later calls return an empty
// batch, which the list treats as "no more".
func (s *Source) RequestMore() parser.RowBatch {
	if s.served {
		return parser.RowBatch{}
	}
	s.served = true
	return s.batch
}

func (s *Source) Close() {}

// HeaderIndices are the row indices carrying a day header, newest first.
func (s *Source) HeaderIndices() []int { return s.headerIndices }
