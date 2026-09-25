package layout

import (
	"math"

	"github.com/luytbq/flowcast/num"
)

// Shape is the primitive shape of an element. Result and writers speak in
// primitive shapes, not semantic types, so writers need not know about kinds.
type Shape string

const (
	ShapeRect          Shape = "rect"
	ShapeDiamond       Shape = "diamond"
	ShapeEllipse       Shape = "ellipse"
	ShapeDoubleEllipse Shape = "double-ellipse"
	ShapeDashedEllipse Shape = "dashed-ellipse"
	ShapeCylinder      Shape = "cylinder"
	ShapeNote          Shape = "note"
)

// NodeKind is the geometry declaration of an element type: the engine and
// writers read only this declaration and never compare type names. Adding an
// element type means adding one line to Kinds; the new type's size and wrapping
// fall back to the defaults in SizeItem unless declared separately.
//
// The outgoing wire rule is shared by all types, so it does not live here:
// outgoing wires leave only from the left, right or bottom side, never the top.
type NodeKind struct {
	Shape Shape
	// EntryTopOnly: incoming wires connect only at the top. The engine does not
	// place the element on the same row as its source and does not let wires
	// enter a lateral side horizontally.
	EntryTopOnly bool
}

// Kinds declares the geometry of each element type.
var Kinds = map[string]NodeKind{
	"task":      {Shape: ShapeRect},
	"condition": {Shape: ShapeDiamond, EntryTopOnly: true},
	"start":     {Shape: ShapeEllipse},
	"end":       {Shape: ShapeDoubleEllipse},
	"external":  {Shape: ShapeDashedEllipse},
	"db":        {Shape: ShapeCylinder},
	"text":      {Shape: ShapeNote},
}

// kindOf returns the declaration of a type; an undeclared type counts as a rectangle.
func kindOf(kind string) NodeKind {
	if k, ok := Kinds[kind]; ok {
		return k
	}
	return NodeKind{Shape: ShapeRect}
}

// singlePort reports whether this shape has only one connection point, at the
// middle of each side. A slanted or curved outline has no straight stretch to
// split into ports like a rectangle's side, so several wires on the same side
// must converge on that side's vertex unless they are forced apart.
func (s Shape) singlePort() bool {
	switch s {
	case ShapeDiamond, ShapeEllipse, ShapeDoubleEllipse, ShapeDashedEllipse:
		return true
	}
	return false
}

// outline is the point on the shape's outline, on side side, at position t
// along that side, as fractions of the bounding box. For a rectangle the point
// lies right on the edge; for diamonds and ellipses the point moves inward from
// the bounding box until it meets the outline, so draw.io draws the wire end
// exactly where it was computed. Rounded to two digits as the writer outputs
// it, so that merge reading the file back sees ports matching the computed ones.
func (s Shape) outline(side byte, t float64) [2]float64 {
	inset := 0.0
	switch s {
	case ShapeDiamond:
		inset = math.Abs(t - 0.5)
	case ShapeEllipse, ShapeDoubleEllipse, ShapeDashedEllipse:
		d := float64(2*t) - 1
		inset = num.Round(0.5-float64(0.5*math.Sqrt(1-float64(d*d))), 2)
	}
	far := num.Round(1-inset, 2)
	switch side {
	case 'B':
		return [2]float64{t, far}
	case 'T':
		return [2]float64{t, inset}
	case 'R':
		return [2]float64{far, t}
	}
	return [2]float64{inset, t}
}
