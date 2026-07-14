package barcodecty

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// externDeclRE matches a top-level declaration in externs.cty. The file is parsed here
// with a regex rather than with functy on purpose: this package must not depend on
// functy (its bytes are opaque to it), and the checks below only need names.
var externDeclRE = regexp.MustCompile(`(?m)^func (\w+)\(`)

// externAttrRE matches an attribute of the options object as declared in externs.cty:
// `    scale            = optional(number),`
var externAttrRE = regexp.MustCompile(`(?m)^\s+(\w+)\s+= optional\(`)

// TestExternsCoverEveryFunction is the drift guard on the name set.
func TestExternsCoverEveryFunction(t *testing.T) {
	declared := make(map[string]bool)
	for _, m := range externDeclRE.FindAllStringSubmatch(string(Externs()), -1) {
		declared[m[1]] = true
	}

	funcs := GetBarcodeFunctions()
	for name := range funcs {
		assert.True(t, declared[name],
			"%s() is provided by GetBarcodeFunctions but has no declaration in externs.cty", name)
	}
	for name := range declared {
		assert.Contains(t, funcs, name,
			"externs.cty declares %s(), which GetBarcodeFunctions does not provide", name)
	}
}

// The options object is the whole reason this package carries an extern: cty declares it
// dynamic, so the declaration is the only place its attributes are named. `validOptions`
// is what actually gets enforced at call time, and the two must not drift — an option
// accepted but undeclared is invisible, and one declared but rejected is a lie.
func TestExternDeclaresEveryOption(t *testing.T) {
	declared := make(map[string]bool)
	for _, m := range externAttrRE.FindAllStringSubmatch(string(Externs()), -1) {
		declared[m[1]] = true
	}

	for name := range validOptions {
		assert.True(t, declared[name],
			"barcode() accepts the option %q, but externs.cty does not declare it — "+
				"so nothing tells a caller it exists", name)
	}
	for name := range declared {
		assert.True(t, validOptions[name],
			"externs.cty declares the option %q, which barcode() rejects as unknown", name)
	}
}

// Every symbology the encoders map can actually produce must be named in the `type`
// parameter's documentation — in the extern, and in the cty metadata a non-functy host
// sees. Adding an encoder without listing it leaves a working symbology nobody knows about.
func TestEverySymbologyIsDocumented(t *testing.T) {
	externs := string(Externs())
	ctyDoc := BarcodeFunc.Params()[0].Description
	require.NotEmpty(t, ctyDoc)

	for name := range encoders {
		quoted := `"` + name + `"`
		assert.Contains(t, externs, quoted,
			"barcode() can encode %s, but externs.cty does not list it", quoted)
		assert.Contains(t, ctyDoc, quoted,
			"barcode() can encode %s, but the cty description of the `type` parameter "+
				"does not list it — which is all a non-functy host can see", quoted)
	}
}

// The bytes must declare themselves an extern file: functy's RegisterExterns verifies the
// directive rather than forcing the mode, so that this same file is a valid standalone
// .cty that `functy fmt` and `functy symbols` can open.
func TestExternsCarryTheDirective(t *testing.T) {
	require.True(t, strings.HasPrefix(string(Externs()), "//functy:extern\n"),
		"externs.cty must begin with the //functy:extern directive")
}

// The cty metadata is the only documentation a non-functy cty host can see, and the only
// thing functy's own doc() reads (doc() does not consult the extern), so a gap here reads
// as "exists but undocumented" even where help() shows a full block.
func TestEverythingIsDescribed(t *testing.T) {
	for name, fn := range GetBarcodeFunctions() {
		assert.NotEmpty(t, fn.Description(), "%s() has no cty Description", name)

		for _, p := range fn.Params() {
			assert.NotEmpty(t, p.Description, "%s() parameter %q has no Description", name, p.Name)
		}
		if vp := fn.VarParam(); vp != nil {
			assert.NotEmpty(t, vp.Description, "%s() variadic parameter %q has no Description", name, vp.Name)
		}
	}
}
