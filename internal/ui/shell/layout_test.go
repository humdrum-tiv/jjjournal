package shell

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/require"
)

func all() map[PanelID]bool {
	return map[PanelID]bool{PanelStatus: true, PanelFiles: true, PanelBookmarks: true,
		PanelJournal: true, PanelDetail: true, PanelNotes: true}
}

func TestAccordionFocusedHalfJournalTwoFifths(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	require.Equal(t, []int{10, 10, 20}, Accordion(40, left, PanelJournal))
	require.Equal(t, []int{20, 4, 16}, Accordion(40, left, PanelFiles))
	require.Equal(t, []int{10, 10, 20}, Accordion(40, left, PanelDetail), "focus elsewhere: Journal half")
	require.Equal(t, []int{40}, Accordion(40, []PanelID{PanelJournal}, PanelFiles))
}

func TestAccordionKeepsMinimumRows(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	require.Equal(t, []int{5, 3, 4}, Accordion(12, left, PanelFiles))
	require.Equal(t, []int{2, 2, 3}, Accordion(7, left, PanelFiles), "below 3 rows each: equal split")
}

func TestAccordionAlwaysSumsToHeight(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	for h := 0; h <= 80; h++ {
		for _, f := range []PanelID{PanelFiles, PanelBookmarks, PanelJournal, PanelDetail} {
			sum := 0
			for _, v := range Accordion(h, left, f) {
				require.GreaterOrEqual(t, v, 0)
				sum += v
			}
			require.Equal(t, h, sum, "h=%d focus=%d", h, f)
		}
	}
}

func TestComputeAllPanels(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	lay := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelJournal, Present: all()})

	require.Equal(t, 68, lay[PanelStatus].R.Dx())
	require.Equal(t, StatusHeight, lay[PanelStatus].R.Dy())
	require.Equal(t, 45, lay[PanelFiles].R.Dy()+lay[PanelBookmarks].R.Dy()+lay[PanelJournal].R.Dy())
	require.Equal(t, 22, lay[PanelJournal].R.Dy(), "focused Journal: half of 45")
	require.Equal(t, 132, lay[PanelDetail].R.Dx())
	require.Equal(t, 10, lay[PanelNotes].R.Dy(), "8 rows + border")
	require.Equal(t, 40, lay[PanelDetail].R.Dy())
	require.Equal(t, 40, lay[PanelNotes].R.Min.Y)
}

func TestComputeAbsentPanelsTakeNoSpace(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	present := map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
	lay := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelJournal, Present: present})

	_, hasFiles := lay[PanelFiles]
	_, hasNotes := lay[PanelNotes]
	require.False(t, hasFiles)
	require.False(t, hasNotes)
	require.Equal(t, 45, lay[PanelJournal].R.Dy())
	require.Equal(t, 50, lay[PanelDetail].R.Dy())
}

func TestComputeZoom(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	col := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelFiles, Zoom: ZoomColumn, Present: all()})
	require.Equal(t, layout.Rect(0, 0, 68, 50), col[PanelFiles].R)
	_, hasJournal := col[PanelJournal]
	require.False(t, hasJournal, "column zoom hides the other left panels")
	require.Equal(t, 40, col[PanelDetail].R.Dy(), "right column unaffected")

	full := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelNotes, Zoom: ZoomFull, Present: all()})
	require.Len(t, full, 1)
	require.Equal(t, box.R, full[PanelNotes].R)
}

func TestComputeClampsLeftWidth(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	present := map[PanelID]bool{PanelJournal: true, PanelDetail: true}
	require.Equal(t, 40, Compute(box, Params{LeftWidth: 0.05, Present: present})[PanelJournal].R.Dx())
	require.Equal(t, 120, Compute(box, Params{LeftWidth: 0.9, Present: present})[PanelJournal].R.Dx())
}

func TestCycleSkipsAbsentPanels(t *testing.T) {
	present := map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
	require.Equal(t, PanelJournal, Cycle(present, PanelStatus, 1))
	require.Equal(t, PanelDetail, Cycle(present, PanelJournal, 1))
	require.Equal(t, PanelStatus, Cycle(present, PanelDetail, 1))
	require.Equal(t, PanelDetail, Cycle(present, PanelStatus, -1))
	require.Equal(t, PanelJournal, Cycle(present, PanelFiles, 1), "unknown start falls back to Journal")
	require.Equal(t, PanelFiles, Cycle(all(), PanelStatus, 1))
}

func TestZoomStepClamps(t *testing.T) {
	require.Equal(t, ZoomColumn, ZoomNone.Step(1))
	require.Equal(t, ZoomFull, ZoomFull.Step(1))
	require.Equal(t, ZoomNone, ZoomNone.Step(-1))
	require.Equal(t, ZoomColumn, ZoomFull.Step(-1))
}
