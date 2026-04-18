# barcode-cty-func

[![CI](https://github.com/tsarna/barcode-cty-func/actions/workflows/ci.yml/badge.svg)](https://github.com/tsarna/barcode-cty-func/actions/workflows/ci.yml)

A Go module providing barcode image generation as a [go-cty](https://github.com/zclconf/go-cty) / HCL2 function. Supports 11 barcode formats and returns PNG images as `bytes` objects.

Backed by [github.com/boombuler/barcode](https://github.com/boombuler/barcode).

## Installation

```
go get github.com/tsarna/barcode-cty-func
```

## Usage

```go
import (
    barcodecty "github.com/tsarna/barcode-cty-func"
    "github.com/zclconf/go-cty/cty/function"
)

// Register all functions in an HCL eval context
funcs := barcodecty.GetBarcodeFunctions()
// funcs is map[string]function.Function — merge into your eval context
```

## Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `barcode` | `barcode(type string, data string[, options object]) bytes` | Generates a barcode image as a PNG `bytes` object |

### `barcode(type, data[, options])`

Generates a barcode image and returns it as a `bytes` object with `content_type = "image/png"`.

```hcl
img = barcode("qr", "https://example.com")
# img.content_type == "image/png"
```

### Barcode Types

| `type` string | Format | Dimensions | Notes |
|---------------|--------|------------|-------|
| `"qr"` | QR Code | 2D | Supports `error_correction` option |
| `"datamatrix"` | Data Matrix | 2D | |
| `"aztec"` | Aztec Code | 2D | |
| `"pdf417"` | PDF 417 | 2D (stacked) | |
| `"code128"` | Code 128 | 1D | Most versatile 1D format; encodes full ASCII |
| `"code93"` | Code 93 | 1D | |
| `"code39"` | Code 39 | 1D | |
| `"codabar"` | Codabar | 1D | Numeric + limited symbols |
| `"ean13"` | EAN-13 | 1D | Exactly 12 digits (check digit appended) |
| `"ean8"` | EAN-8 | 1D | Exactly 7 digits (check digit appended) |
| `"2of5"` | Interleaved 2-of-5 | 1D | Numeric only; even number of digits required |

### Options

The optional third argument is an object. All fields are optional. `scale` and `width` are mutually exclusive (both control horizontal sizing). `height` can be used alone or combined with either `scale` or `width`.

| Field | Type | Default | Applies to | Description |
|-------|------|---------|------------|-------------|
| `scale` | number | `4` | all | Integer pixel multiplier applied to the barcode's natural symbol size |
| `width` | number | — | all | Output image width in pixels (requires `height`; mutually exclusive with `scale`) |
| `height` | number | see below | all | Output image height in pixels |
| `error_correction` | string | `"M"` | `"qr"` only | Error correction level: `"L"`, `"M"`, `"Q"`, or `"H"` |

**Sizing behaviour:**

- **`scale`**: multiplies the barcode's natural width by the given integer. For 2D codes, height is scaled equally. For 1D codes, height is set to a sensible default (see below). Every module/bar maps to exactly `scale` pixels wide with no interpolation artefacts.
- **`height`**: sets an explicit pixel height. Can be combined with `scale` (which controls width) or used alone (with default `scale = 4` for width).
- **`width` / `height`**: both dimensions set explicitly. Useful when the image must fit a specific pixel budget.
- **Neither specified**: defaults to `scale = 4`.

**Default 1D barcode height:**

1D barcodes have a natural height of only 1 pixel, so the function applies a default height when none is specified:

- **Code 128**: 20% of the scaled width, producing a proportional barcode regardless of data length.
- **All other 1D types**: `scale * 24` (96px at the default scale of 4).

These defaults can always be overridden with an explicit `height` option.

## Examples

### QR code with defaults

```hcl
img = barcode("qr", "https://example.com")
```

### QR code with high error correction and explicit scale

```hcl
img = barcode("qr", "https://example.com/p?id=${ctx.payload.id}", {
    scale            = 8,
    error_correction = "H",
})
```

### QR code at a fixed pixel size

```hcl
img = barcode("qr", "https://example.com", {
    width  = 400,
    height = 400,
})
```

### Code 128 label

```hcl
label = barcode("code128", ctx.payload.tracking_number, {
    scale = 3,
})
```

### 1D barcode with explicit height

```hcl
label = barcode("code39", "ITEM-42", {
    height = 80,
})
```

### EAN-13 for retail product

```hcl
# data must be exactly 12 digits; check digit is computed and appended
ean = barcode("ean13", ctx.payload.gtin12)
```

## License

[BSD 2-Clause](LICENSE)
