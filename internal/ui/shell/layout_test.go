package shell

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/require"
)

func TestComputeSplitsColumnsAndStatus(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	lay := Compute(box, 0.34)

	require.Equal(t, 68, lay.Status.R.Dx(), "left column is 34% of 200")
	require.Equal(t, 68, lay.Journal.R.Dx())
	require.Equal(t, 132, lay.Detail.R.Dx())
	require.Equal(t, StatusHeight, lay.Status.R.Dy())
	require.Equal(t, 50-StatusHeight, lay.Journal.R.Dy())
	require.Equal(t, 50, lay.Detail.R.Dy())
	require.Equal(t, 0, lay.Status.R.Min.Y)
	require.Equal(t, StatusHeight, lay.Journal.R.Min.Y)
	require.Equal(t, 68, lay.Detail.R.Min.X)
}

func TestComputeClampsLeftWidth(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	require.Equal(t, 40, Compute(box, 0.05).Journal.R.Dx(), "floor 20%")
	require.Equal(t, 120, Compute(box, 0.9).Journal.R.Dx(), "ceiling 60%")
}

func TestPanelCycleOrder(t *testing.T) {
	require.Equal(t, PanelJournal, PanelStatus.next())
	require.Equal(t, PanelDetail, PanelJournal.next())
	require.Equal(t, PanelStatus, PanelDetail.next())
	require.Equal(t, PanelDetail, PanelStatus.prev())
	require.Equal(t, PanelJournal, PanelFiles.next(), "panels not in this plan fall back to Journal")
}
