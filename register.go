package barcodecty

import "github.com/zclconf/go-cty/cty/function"

// GetBarcodeFunctions returns the barcode function set for registration
// in a cty evaluation context.
func GetBarcodeFunctions() map[string]function.Function {
	return map[string]function.Function{
		"barcode": BarcodeFunc,
	}
}
