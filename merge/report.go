package merge

import (
	"strconv"
	"strings"
)

// Report lists what merge did, so the user knows what needs a second look.
type Report struct {
	Pinned           []string // nodes that kept their old position
	Placed           []string // new nodes placed
	Shifted          []string // new nodes shifted down because they overlapped something
	LaneChanged      []string // nodes that changed lane, re-placed like new nodes
	KeptEdges        []string // wires that kept their old waypoints
	Rerouted         []string // wires left for draw.io to route
	LabelsCentered   []string // manually edited wires, label moved to the midpoint
	Removed          []string // tool cells no longer in the table
	LanesGrown       []string // "id +Npx"
	FreehandKept     []string
	FreehandDropped  []string // parent was removed
	FreehandDetached []string // "id.source" or "id.target"
	Overlaps         []string // "a/b"
}

// Lines are the lines the CLI prints. Their order and wording are interface,
// pinned by the CLI transcripts.
func (r Report) Lines() []string {
	row := func(label string, xs []string) string {
		list := "-"
		if len(xs) > 0 {
			list = strings.Join(xs, ", ")
		}
		return "merge: " + label + " (" + strconv.Itoa(len(xs)) + "): " + list
	}
	return []string{
		row("kept position", r.Pinned),
		row("new element", r.Placed),
		row("new element shifted down due to overlap", r.Shifted),
		row("lane changed in table or dragged to another lane, re-placed", r.LaneChanged),
		row("edge kept waypoints", r.KeptEdges),
		row("edge left to draw.io routing, needs manual edits", r.Rerouted),
		row("edge manually edited, label moved to midpoint", r.LabelsCentered),
		row("removed, no longer in table", r.Removed),
		row("lane widened", r.LanesGrown),
		row("freehand cell kept", r.FreehandKept),
		row("freehand cell dropped (parent removed)", r.FreehandDropped),
		row("freehand cell lost endpoint", r.FreehandDetached),
		row("overlapping elements", r.Overlaps),
	}
}
