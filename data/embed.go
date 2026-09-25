// Package data embeds the shared data tables into the binary.
//
// The font metrics table lives here instead of being read from a path at run
// time, so flowcast is a single executable that runs anywhere.
package data

import _ "embed"

// Verdana is the advance width table of Verdana, produced by tools/extractmetrics.
//
//go:embed verdana.json
var Verdana []byte
