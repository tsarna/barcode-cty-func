package barcodecty

import (
	"bytes"
	"image/png"
	"testing"

	bytescty "github.com/tsarna/bytes-cty-type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// callBarcode is a helper that calls BarcodeFunc with the given type, data, and optional options.
func callBarcode(t *testing.T, btype, data string, opts ...cty.Value) cty.Value {
	t.Helper()
	args := []cty.Value{cty.StringVal(btype), cty.StringVal(data)}
	args = append(args, opts...)
	result, err := BarcodeFunc.Call(args)
	require.NoError(t, err)
	return result
}

// assertValidPNG extracts bytes from a result and verifies it decodes as a valid PNG.
func assertValidPNG(t *testing.T, result cty.Value) {
	t.Helper()
	assert.Equal(t, bytescty.BytesObjectType, result.Type())
	assert.Equal(t, "image/png", result.GetAttr("content_type").AsString())
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	_, err = png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
}

// --- All barcode types ---

func TestBarcode_AllTypes(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"qr", "https://example.com"},
		{"datamatrix", "hello"},
		{"aztec", "hello"},
		{"pdf417", "hello"},
		{"code128", "Hello123"},
		{"code93", "HELLO"},
		{"code39", "HELLO"},
		{"codabar", "A12345B"},
		{"ean13", "590123412345"},
		{"ean8", "5512345"},
		{"2of5", "1234567890"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callBarcode(t, tt.name, tt.data)
			assertValidPNG(t, result)
		})
	}
}

// --- Default scale ---

func TestBarcode_DefaultScale(t *testing.T) {
	result := callBarcode(t, "qr", "test")
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	// Default scale=4, so dimensions should be multiples of 4.
	assert.Equal(t, 0, img.Bounds().Dx()%4)
	assert.Equal(t, 0, img.Bounds().Dy()%4)
}

// --- Scale option ---

func TestBarcode_ScaleOption(t *testing.T) {
	result := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
		"scale": cty.NumberIntVal(8),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 0, img.Bounds().Dx()%8)
	assert.Equal(t, 0, img.Bounds().Dy()%8)
}

// --- Width/height option ---

func TestBarcode_WidthHeight(t *testing.T) {
	result := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
		"width":  cty.NumberIntVal(400),
		"height": cty.NumberIntVal(400),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 400, img.Bounds().Dx())
	assert.Equal(t, 400, img.Bounds().Dy())
}

// --- QR error correction levels ---

func TestBarcode_QR_ErrorCorrectionLevels(t *testing.T) {
	for _, level := range []string{"L", "M", "Q", "H"} {
		t.Run(level, func(t *testing.T) {
			result := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
				"error_correction": cty.StringVal(level),
			}))
			assertValidPNG(t, result)
		})
	}
}

// --- Empty options object uses defaults ---

func TestBarcode_EmptyOptions(t *testing.T) {
	result := callBarcode(t, "qr", "test", cty.EmptyObjectVal)
	assertValidPNG(t, result)
}

// --- Registration ---

func TestGetBarcodeFunctions(t *testing.T) {
	funcs := GetBarcodeFunctions()
	assert.Contains(t, funcs, "barcode")
	assert.Len(t, funcs, 1)
}

// --- Error cases ---

func TestBarcode_Error_UnknownType(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("unknown"),
		cty.StringVal("data"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown barcode type "unknown"`)
	assert.Contains(t, err.Error(), "valid types are:")
}

func TestBarcode_Error_ErrorCorrectionNonQR(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("code128"),
		cty.StringVal("hello"),
		cty.ObjectVal(map[string]cty.Value{
			"error_correction": cty.StringVal("H"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `error_correction option is only valid for type "qr"`)
}

func TestBarcode_Error_ScaleAndWidth(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.ObjectVal(map[string]cty.Value{
			"scale":  cty.NumberIntVal(4),
			"width":  cty.NumberIntVal(400),
			"height": cty.NumberIntVal(400),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scale and width are mutually exclusive")
}

func TestBarcode_Error_ScaleNotPositive(t *testing.T) {
	tests := []struct {
		name  string
		value cty.Value
	}{
		{"zero", cty.NumberIntVal(0)},
		{"negative", cty.NumberIntVal(-1)},
		{"float", cty.NumberFloatVal(1.5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BarcodeFunc.Call([]cty.Value{
				cty.StringVal("qr"),
				cty.StringVal("test"),
				cty.ObjectVal(map[string]cty.Value{
					"scale": tt.value,
				}),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "scale must be a positive integer")
		})
	}
}

func TestBarcode_Error_WidthWithoutHeight(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.ObjectVal(map[string]cty.Value{
			"width": cty.NumberIntVal(400),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "width requires height")
}

func TestBarcode_HeightAlone(t *testing.T) {
	// height alone is valid — width comes from scale (default 4).
	result := callBarcode(t, "code128", "HELLO", cty.ObjectVal(map[string]cty.Value{
		"height": cty.NumberIntVal(100),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 100, img.Bounds().Dy())
	// Width should be natural width * default scale (4).
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestBarcode_ScaleAndHeight(t *testing.T) {
	// scale + height is valid — scale controls width, height is explicit.
	result := callBarcode(t, "code128", "HELLO", cty.ObjectVal(map[string]cty.Value{
		"scale":  cty.NumberIntVal(2),
		"height": cty.NumberIntVal(80),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 80, img.Bounds().Dy())
}

func TestBarcode_Error_WidthHeightNotPositive(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.ObjectVal(map[string]cty.Value{
			"width":  cty.NumberIntVal(0),
			"height": cty.NumberIntVal(400),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "width and height must be positive integers")
}

func TestBarcode_Error_UnknownOption(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.ObjectVal(map[string]cty.Value{
			"bogus": cty.StringVal("nope"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown barcode option "bogus"`)
}

func TestBarcode_Error_TooManyArgs(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.EmptyObjectVal,
		cty.EmptyObjectVal,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "barcode() takes 2 or 3 arguments")
}

func TestBarcode_Error_InvalidData(t *testing.T) {
	// EAN-13 requires exactly 12 digits.
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("ean13"),
		cty.StringVal("notdigits"),
	})
	require.Error(t, err)
}

func TestBarcode_Error_InvalidErrorCorrection(t *testing.T) {
	_, err := BarcodeFunc.Call([]cty.Value{
		cty.StringVal("qr"),
		cty.StringVal("test"),
		cty.ObjectVal(map[string]cty.Value{
			"error_correction": cty.StringVal("X"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid error_correction "X"`)
}

// --- 1D default height ---

func TestBarcode_1D_DefaultHeight(t *testing.T) {
	// 1D codes at default scale=4 should have height = 4 * 24 = 96.
	result := callBarcode(t, "codabar", "A12345B")
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 96, img.Bounds().Dy())
}

func TestBarcode_1D_ScaledHeight(t *testing.T) {
	// 1D codes at scale=2 should have height = 2 * 24 = 48.
	result := callBarcode(t, "codabar", "A12345B", cty.ObjectVal(map[string]cty.Value{
		"scale": cty.NumberIntVal(2),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	assert.Equal(t, 48, img.Bounds().Dy())
}

func TestBarcode_Code128_HeightRatio(t *testing.T) {
	// Code128 default height should be 20% of its scaled width.
	result := callBarcode(t, "code128", "HELLO")
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	assert.Equal(t, int(float64(w)*0.20), h)
}

func TestBarcode_2D_SymmetricScale(t *testing.T) {
	// 2D codes should still scale both dimensions equally.
	result := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
		"scale": cty.NumberIntVal(2),
	}))
	b, err := bytescty.GetBytesFromValue(result)
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(b.Data))
	require.NoError(t, err)
	// QR codes are square, so width == height.
	assert.Equal(t, img.Bounds().Dx(), img.Bounds().Dy())
}

// --- PNG dimensions with scale ---

func TestBarcode_ScaleDimensions(t *testing.T) {
	// Generate QR at scale=1 and scale=3, verify dimensions are exactly 1x and 3x.
	result1 := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
		"scale": cty.NumberIntVal(1),
	}))
	b1, err := bytescty.GetBytesFromValue(result1)
	require.NoError(t, err)
	img1, err := png.Decode(bytes.NewReader(b1.Data))
	require.NoError(t, err)
	naturalW := img1.Bounds().Dx()
	naturalH := img1.Bounds().Dy()

	result3 := callBarcode(t, "qr", "test", cty.ObjectVal(map[string]cty.Value{
		"scale": cty.NumberIntVal(3),
	}))
	b3, err := bytescty.GetBytesFromValue(result3)
	require.NoError(t, err)
	img3, err := png.Decode(bytes.NewReader(b3.Data))
	require.NoError(t, err)
	assert.Equal(t, naturalW*3, img3.Bounds().Dx())
	assert.Equal(t, naturalH*3, img3.Bounds().Dy())
}
