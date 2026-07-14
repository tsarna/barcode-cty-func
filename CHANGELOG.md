# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-07-14

### Added

- **`Externs()` — the real signature of `barcode`, for functy hosts.** `externs.cty`
  (embedded; exposed as opaque bytes via `Externs()` and `ExternsFilename`) declares
  what `barcode` actually accepts, as a [functy](https://github.com/tsarna/functy)
  `//functy:extern` declaration. It is never compiled and declares nothing callable; it
  exists so that `help()`, generated documentation, and editor tooling can show what the
  cty metadata cannot:

  The options object is *optional*, and the only way cty offers to make an argument
  optional is to make it variadic — which claims it may be repeated, when it may not.
  cty also has no way to describe the object's shape, so it is `dynamic`. Reflected from
  cty alone the function read as `barcode(type, data, ...options)`, with `scale`,
  `width`, `height`, and `error_correction` all invisible. The declaration names them.

  This package does not import functy; the bytes are opaque to it. A host registers them:

  ```go
  parser.RegisterExterns(barcodecty.Externs(), barcodecty.ExternsFilename)
  ```

### Changed

- The function and its parameters now carry cty `Description`s — the `type` and `data`
  parameters, and the options variadic (which lists every option and its rules). The
  metadata is the only documentation a non-functy cty host can see, and the only thing
  functy's own `doc()` reads.
- Depends on `bytes-cty-type` v0.2.0 (was v0.1.0).

## [0.1.0] - 2026-04-17

### Added

- Initial release: `barcode(type, data[, options])`, generating any of 11 symbologies
  (QR, Data Matrix, Aztec, PDF417, Code 128/93/39, Codabar, EAN-13/8, Interleaved 2-of-5)
  as a PNG `bytes` value.
