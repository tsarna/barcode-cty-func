package barcodecty

import (
	"bytes"
	"fmt"
	"image/png"
	"sort"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/codabar"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/code93"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
	bytescty "github.com/tsarna/bytes-cty-type"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

const defaultBarHeight = 24 // multiplied by scale for default 1D barcode height

// defaultHeightRatio maps barcode types to a default height expressed as a
// fraction of the scaled width. Types not in this map use scale * defaultBarHeight.
var defaultHeightRatio = map[string]float64{
	"code128": 0.20,
}

type barcodeOptions struct {
	scale           int
	width           int
	height          int
	errorCorrection string
	hasWidth        bool
	hasHeight       bool
}

var is1D = map[string]bool{
	"code128": true,
	"code93":  true,
	"code39":  true,
	"codabar": true,
	"ean13":   true,
	"ean8":    true,
	"2of5":    true,
}

var qrErrorCorrectionLevels = map[string]qr.ErrorCorrectionLevel{
	"L": qr.L,
	"M": qr.M,
	"Q": qr.Q,
	"H": qr.H,
}

var validOptions = map[string]bool{
	"scale":            true,
	"width":            true,
	"height":           true,
	"error_correction": true,
}

type encoderFunc func(data string, opts barcodeOptions) (barcode.Barcode, error)

var encoders = map[string]encoderFunc{
	"qr": func(data string, opts barcodeOptions) (barcode.Barcode, error) {
		ecLevel, ok := qrErrorCorrectionLevels[opts.errorCorrection]
		if !ok {
			return nil, fmt.Errorf("invalid error_correction %q; valid values are: \"L\", \"M\", \"Q\", \"H\"", opts.errorCorrection)
		}
		return qr.Encode(data, ecLevel, qr.Auto)
	},
	"datamatrix": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return datamatrix.Encode(data)
	},
	"aztec": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return aztec.Encode([]byte(data), aztec.DEFAULT_EC_PERCENT, aztec.DEFAULT_LAYERS)
	},
	"pdf417": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return pdf417.Encode(data, 2)
	},
	"code128": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return code128.Encode(data)
	},
	"code93": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return code93.Encode(data, true, true)
	},
	"code39": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return code39.Encode(data, true, true)
	},
	"codabar": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return codabar.Encode(data)
	},
	"ean13": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return ean.Encode(data)
	},
	"ean8": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return ean.Encode(data)
	},
	"2of5": func(data string, _ barcodeOptions) (barcode.Barcode, error) {
		return twooffive.Encode(data, true)
	},
}

func validTypeList() string {
	types := make([]string, 0, len(encoders))
	for k := range encoders {
		types = append(types, fmt.Sprintf("%q", k))
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

// BarcodeFunc generates a barcode image and returns it as a bytes object
// with content_type "image/png".
// Called as barcode(type, data) or barcode(type, data, options).
var BarcodeFunc = function.New(&function.Spec{
	Description: "Generates a barcode image as a bytes object with content_type image/png",
	Params: []function.Parameter{
		{Name: "type", Type: cty.String},
		{Name: "data", Type: cty.String},
	},
	VarParam: &function.Parameter{
		Name:             "options",
		Type:             cty.DynamicPseudoType,
		AllowDynamicType: true,
		AllowNull:        true,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 3 {
			return cty.NilType, fmt.Errorf("barcode() takes 2 or 3 arguments")
		}
		return bytescty.BytesObjectType, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		btype := args[0].AsString()
		data := args[1].AsString()

		var optVal cty.Value
		if len(args) > 2 {
			optVal = args[2]
		}

		opts, err := parseBarcodeOptions(optVal, btype)
		if err != nil {
			return cty.NilVal, err
		}

		return encodeBarcode(btype, data, opts)
	},
})

func parseBarcodeOptions(opts cty.Value, btype string) (barcodeOptions, error) {
	o := barcodeOptions{
		errorCorrection: "M",
	}

	if opts == cty.NilVal || opts.IsNull() {
		return o, nil
	}

	ty := opts.Type()
	if !ty.IsObjectType() {
		return o, fmt.Errorf("barcode options must be an object, got %s", ty.FriendlyName())
	}

	// Reject unknown attributes.
	for name := range ty.AttributeTypes() {
		if !validOptions[name] {
			return o, fmt.Errorf("unknown barcode option %q", name)
		}
	}

	hasScale := false
	hasWidth := false
	hasHeight := false

	if ty.HasAttribute("scale") {
		v := opts.GetAttr("scale")
		if !v.IsNull() {
			if v.Type() != cty.Number {
				return o, fmt.Errorf("scale must be a positive integer")
			}
			bf := v.AsBigFloat()
			if !bf.IsInt() {
				return o, fmt.Errorf("scale must be a positive integer")
			}
			n, _ := bf.Int64()
			if n <= 0 {
				return o, fmt.Errorf("scale must be a positive integer")
			}
			o.scale = int(n)
			hasScale = true
		}
	}

	if ty.HasAttribute("width") {
		v := opts.GetAttr("width")
		if !v.IsNull() {
			if v.Type() != cty.Number {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			bf := v.AsBigFloat()
			if !bf.IsInt() {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			n, _ := bf.Int64()
			if n <= 0 {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			o.width = int(n)
			hasWidth = true
		}
	}

	if ty.HasAttribute("height") {
		v := opts.GetAttr("height")
		if !v.IsNull() {
			if v.Type() != cty.Number {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			bf := v.AsBigFloat()
			if !bf.IsInt() {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			n, _ := bf.Int64()
			if n <= 0 {
				return o, fmt.Errorf("width and height must be positive integers")
			}
			o.height = int(n)
			hasHeight = true
		}
	}

	if ty.HasAttribute("error_correction") {
		v := opts.GetAttr("error_correction")
		if !v.IsNull() {
			if v.Type() != cty.String {
				return o, fmt.Errorf("error_correction must be a string")
			}
			o.errorCorrection = v.AsString()
		}
	}

	// scale and width are mutually exclusive (both control horizontal sizing).
	if hasScale && hasWidth {
		return o, fmt.Errorf("scale and width are mutually exclusive")
	}

	// width requires height.
	if hasWidth && !hasHeight {
		return o, fmt.Errorf("width requires height")
	}

	o.hasWidth = hasWidth
	o.hasHeight = hasHeight

	// Validate error_correction only with QR.
	if ty.HasAttribute("error_correction") && btype != "qr" {
		v := opts.GetAttr("error_correction")
		if !v.IsNull() {
			return o, fmt.Errorf("error_correction option is only valid for type \"qr\"")
		}
	}

	return o, nil
}

func encodeBarcode(btype, data string, opts barcodeOptions) (cty.Value, error) {
	encoderFn, ok := encoders[btype]
	if !ok {
		return cty.NilVal, fmt.Errorf("unknown barcode type %q; valid types are: %s", btype, validTypeList())
	}

	bc, err := encoderFn(data, opts)
	if err != nil {
		return cty.NilVal, err
	}

	scale := opts.scale
	if scale == 0 {
		scale = 4
	}

	var w, h int
	if opts.hasWidth {
		// Explicit width and height.
		w = opts.width
		h = opts.height
	} else {
		// Width from scale applied to natural size.
		b := bc.Bounds()
		w = b.Dx() * scale

		if opts.hasHeight {
			// Explicit height, scaled width.
			h = opts.height
		} else if ratio, ok := defaultHeightRatio[btype]; ok {
			// Height as a fraction of the scaled width.
			h = int(float64(w) * ratio)
			if h < 1 {
				h = 1
			}
		} else if is1D[btype] {
			// Default 1D height proportional to scale.
			h = scale * defaultBarHeight
		} else {
			// 2D: scale both dimensions equally.
			h = b.Dy() * scale
		}
	}

	bc, err = barcode.Scale(bc, w, h)
	if err != nil {
		return cty.NilVal, fmt.Errorf("barcode scale error: %s", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, bc); err != nil {
		return cty.NilVal, fmt.Errorf("barcode PNG encoding error: %s", err)
	}

	return bytescty.BuildBytesObject(buf.Bytes(), "image/png"), nil
}
