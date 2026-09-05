package revisions

import (
	"context"
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/parser"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

type fakeSource struct{ closed bool }

func (f *fakeSource) RequestMore() parser.RowBatch { return parser.RowBatch{} }
func (f *fakeSource) Close()                       { f.closed = true }

// failingRunner only implements the method DefaultRowSource reaches.
type failingRunner struct{ appContext.CommandRunner }

func (failingRunner) RunCommandStreaming(context.Context, []string) (*appContext.StreamingCommand, error) {
	return nil, errors.New("boom")
}

func TestLoadStreamingUsesConfiguredRowSourceFactory(t *testing.T) {
	m := New(&appContext.MainContext{JJConfig: &config.JJConfig{}})
	src := &fakeSource{}
	var gotRevset, gotTemplate string
	m.SetRowSourceFactory(func(_ context.Context, _ appContext.CommandRunner, revset, jjTemplate string) (RowSource, error) {
		gotRevset, gotTemplate = revset, jjTemplate
		return src, nil
	})
	tag := m.tag.Add(1)

	msg := m.loadStreaming("all()", tag)()

	ready, ok := msg.(streamingReadyMsg)
	require.True(t, ok, "expected streamingReadyMsg, got %T", msg)
	require.Same(t, src, ready.streamer)
	require.NoError(t, ready.err)
	require.Equal(t, "all()", gotRevset)
	require.Equal(t, "", gotTemplate, "jj's templates.log is empty in this context")
}

func TestLoadStreamingReportsFactoryFailure(t *testing.T) {
	m := New(&appContext.MainContext{JJConfig: &config.JJConfig{}})
	m.SetRowSourceFactory(func(context.Context, appContext.CommandRunner, string, string) (RowSource, error) {
		return nil, errors.New("no rows for you")
	})
	msg := m.loadStreaming("all()", m.tag.Add(1))()
	ready := msg.(streamingReadyMsg)
	require.Nil(t, ready.streamer)
	require.EqualError(t, ready.err, "no rows for you")
	require.Equal(t, "no rows for you", ready.output)
}

func TestDefaultRowSourceReturnsUntypedNilOnFailure(t *testing.T) {
	src, err := DefaultRowSource(context.Background(), failingRunner{}, "all()", "")
	require.Error(t, err)
	require.True(t, src == nil, "a failed source must be an untyped nil, not a nil *GraphStreamer in an interface")
}
