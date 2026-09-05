// Package shell is jjj's lazygit-style layout: several panels visible at once
// around upstream's revisions list. It plugs into ui.Model through the
// ui.Extension hook and only ever sees keys while upstream is in normal mode.
package shell

import "github.com/idursun/jjui/internal/ui/layout"

// PanelID numbers panels the lazygit way (spec §3.1); the digit is the focus key.
type PanelID int

const (
	PanelDetail    PanelID = 0
	PanelStatus    PanelID = 1
	PanelFiles     PanelID = 2 // P2
	PanelBookmarks PanelID = 3 // P2
	PanelJournal   PanelID = 4
	PanelTasks     PanelID = 5 // P3
	PanelNotes     PanelID = 6 // P2
	PanelStats     PanelID = 7 // P4
)

// cycle is the tab order among the panels this plan ships.
var cycle = []PanelID{PanelStatus, PanelJournal, PanelDetail}

func (p PanelID) next() PanelID { return stepPanel(p, 1) }
func (p PanelID) prev() PanelID { return stepPanel(p, -1) }

func stepPanel(p PanelID, d int) PanelID {
	for i, id := range cycle {
		if id == p {
			return cycle[(i+d+len(cycle))%len(cycle)]
		}
	}
	return PanelJournal
}

// Layout holds the outer box (border included) of each panel for one frame.
type Layout struct {
	Status, Journal, Detail layout.Box
}

// StatusHeight is the Status panel's outer height: three content lines plus border.
const StatusHeight = 5

// Compute splits box into a left column (Status above Journal) and a right
// column (Detail). leftWidth is the left column's share, clamped to [0.2, 0.6].
func Compute(box layout.Box, leftWidth float64) Layout {
	if leftWidth < 0.2 {
		leftWidth = 0.2
	}
	if leftWidth > 0.6 {
		leftWidth = 0.6
	}
	cols := box.H(layout.Percent(int(leftWidth*100+0.5)), layout.Fill(1))
	if len(cols) < 2 {
		return Layout{Journal: box}
	}
	rows := cols[0].V(layout.Fixed(StatusHeight), layout.Fill(1))
	if len(rows) < 2 {
		return Layout{Journal: cols[0], Detail: cols[1]}
	}
	return Layout{Status: rows[0], Journal: rows[1], Detail: cols[1]}
}
