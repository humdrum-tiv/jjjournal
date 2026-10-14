// Package shell is jjj's lazygit-style layout: several panels visible at once
// around upstream's revisions list. It plugs into ui.Model through the
// ui.Extension hook; its keys reach it as intents through the scopes it
// contributes (see Shell.Scopes).
package shell

import (
	"slices"

	"github.com/idursun/jjui/internal/ui/layout"
)

// PanelID numbers panels the lazygit way (spec §3.1); the digit is the focus key.
type PanelID int

const (
	PanelDetail    PanelID = 0
	PanelStatus    PanelID = 1
	PanelFiles     PanelID = 2
	PanelBookmarks PanelID = 3
	PanelJournal   PanelID = 4
	PanelTasks     PanelID = 5 // P3
	PanelNotes     PanelID = 6
	PanelStats     PanelID = 7 // P4
)

// Order is the draw order and the tab order.
var Order = []PanelID{PanelStatus, PanelFiles, PanelBookmarks, PanelJournal, PanelDetail, PanelNotes}

// leftFlex are the left column's accordion panels, top to bottom.
var leftFlex = []PanelID{PanelFiles, PanelBookmarks, PanelJournal}

func isLeft(id PanelID) bool { return id == PanelStatus || slices.Contains(leftFlex, id) }

// Zoom is the focused panel's zoom level (spec §4.1): + steps up, _ down.
type Zoom int

const (
	ZoomNone Zoom = iota
	ZoomColumn
	ZoomFull
)

func (z Zoom) Step(d int) Zoom { return Zoom(max(int(ZoomNone), min(int(ZoomFull), int(z)+d))) }

const (
	// StatusHeight is the Status panel's outer height: three lines plus border.
	StatusHeight = 5
	// MinPanelRows is the smallest outer height an accordion panel gets
	// while the column has room for every panel at that size.
	MinPanelRows = 3
)

type Params struct {
	LeftWidth float64
	NotesRows int // Notes' inner rows; its outer height adds the border
	Focus     PanelID
	Zoom      Zoom
	Present   map[PanelID]bool // registered and not hidden
}

// Layout maps each panel drawn this frame to its outer box (border included).
type Layout map[PanelID]layout.Box

// Compute splits box into a left column (Status, then the accordion panels)
// and a right column (Detail above Notes). Zoom gives the focused panel its
// whole column or the whole box. Absent panels take no space.
func Compute(box layout.Box, p Params) Layout {
	lay := Layout{}
	if p.Zoom == ZoomFull {
		lay[p.Focus] = box
		return lay
	}
	lw := min(max(p.LeftWidth, 0.2), 0.6)
	cols := box.H(layout.Percent(int(lw*100+0.5)), layout.Fill(1))
	if len(cols) < 2 {
		lay[PanelJournal] = box
		return lay
	}
	left, right := cols[0], cols[1]
	if p.Zoom == ZoomColumn && isLeft(p.Focus) {
		lay[p.Focus] = left
	} else {
		computeLeft(lay, left, p)
	}
	if p.Zoom == ZoomColumn && !isLeft(p.Focus) {
		lay[p.Focus] = right
	} else {
		computeRight(lay, right, p)
	}
	return lay
}

func computeLeft(lay Layout, col layout.Box, p Params) {
	rest := col
	if p.Present[PanelStatus] {
		var top layout.Box
		top, rest = col.CutTop(StatusHeight)
		lay[PanelStatus] = top
	}
	var panels []PanelID
	for _, id := range leftFlex {
		if id == PanelJournal || p.Present[id] {
			panels = append(panels, id)
		}
	}
	for i, h := range Accordion(rest.R.Dy(), panels, p.Focus) {
		var b layout.Box
		b, rest = rest.CutTop(h)
		lay[panels[i]] = b
	}
}

func computeRight(lay Layout, col layout.Box, p Params) {
	notes := p.NotesRows + 2
	if p.Present[PanelNotes] && p.NotesRows > 0 && col.R.Dy() >= notes+MinPanelRows {
		detail, n := col.CutBottom(notes)
		lay[PanelDetail], lay[PanelNotes] = detail, n
		return
	}
	lay[PanelDetail] = col
}

// Accordion splits h rows among panels (top to bottom). The focused panel
// gets half; an unfocused Journal gets two fifths, or half when focus is
// outside these panels; the others share the rest with at least
// MinPanelRows each, taking rows back from the largest panel if needed.
// Below MinPanelRows per panel it splits equally. The result sums to h.
func Accordion(h int, panels []PanelID, focus PanelID) []int {
	n := len(panels)
	out := make([]int, n)
	switch {
	case n == 0:
		return out
	case n == 1:
		out[0] = h
		return out
	case h < n*MinPanelRows:
		for i := range out {
			out[i] = h / n
		}
		out[n-1] += h - (h/n)*n
		return out
	}

	assigned := make([]bool, n)
	fi, ji := slices.Index(panels, focus), slices.Index(panels, PanelJournal)
	if fi >= 0 {
		out[fi], assigned[fi] = h/2, true
	}
	if ji >= 0 && ji != fi {
		if fi >= 0 {
			out[ji] = h * 2 / 5
		} else {
			out[ji] = h / 2
		}
		assigned[ji] = true
	}

	used, others := 0, 0
	for i := range out {
		used += out[i]
		if !assigned[i] {
			others++
		}
	}
	rem := h - used
	if others == 0 {
		target := ji
		if target < 0 {
			target = fi
		}
		out[target] += rem
		return out
	}
	share := rem / others
	if share < MinPanelRows {
		for need := MinPanelRows*others - rem; need > 0; need-- {
			out[largestAssigned(out, assigned)]--
		}
		rem, share = MinPanelRows*others, MinPanelRows
	}
	last := -1
	for i := range out {
		if !assigned[i] {
			out[i], last = share, i
		}
	}
	out[last] += rem - share*others
	return out
}

func largestAssigned(out []int, assigned []bool) int {
	best := -1
	for i := range out {
		if assigned[i] && (best < 0 || out[i] > out[best]) {
			best = i
		}
	}
	return best
}

// Cycle steps from `from` by d through the present panels in Order.
// An unknown or absent start returns PanelJournal.
func Cycle(present map[PanelID]bool, from PanelID, d int) PanelID {
	var ids []PanelID
	for _, id := range Order {
		if present[id] {
			ids = append(ids, id)
		}
	}
	i := slices.Index(ids, from)
	if i < 0 || len(ids) == 0 {
		return PanelJournal
	}
	return ids[((i+d)%len(ids)+len(ids))%len(ids)]
}
