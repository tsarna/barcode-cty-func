package barcodecty

import _ "embed"

//go:embed externs.cty
var externsCty []byte

// ExternsFilename is the name reported for the embedded declarations in
// diagnostics.
const ExternsFilename = "barcode-cty-func/externs.cty"

// Externs returns the functy `//functy:extern` declaration for the function
// GetBarcodeFunctions provides: its real signature, which its cty metadata cannot
// express.
//
// The options object is *optional*, and the only way cty offers to make an argument
// optional is to make it variadic — which says it may be repeated, when it may not.
// cty also has no way to describe the object's shape, so it is declared dynamic:
// reflected from cty alone the function reads as `barcode(type, data, ...options)`, and
// what may go in that object is anybody's guess. The declaration gives it a name, a
// type, and its four attributes, so that help(), generated documentation, and editor
// tooling can show them.
//
// The bytes are opaque to this package: it does not import functy, and nothing here
// parses them. A functy host registers them:
//
//	parser.RegisterExterns(barcodecty.Externs(), barcodecty.ExternsFilename)
func Externs() []byte { return externsCty }
