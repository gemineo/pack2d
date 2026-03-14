# pack2d

A Go library that compresses and encodes textual payloads into compact strings optimized for 2D barcodes (QR Code, DataMatrix), and decodes them back. It also generates barcode images.

```
ENCODE:  text → [serialize] → compress → base45 → barcode image
DECODE:  base45 string → decompress → [deserialize] → text
```

Base45 is used (instead of Base64) because its alphabet maps exactly to the QR alphanumeric character set, enabling the more compact QR alphanumeric encoding mode.

## Installation

```bash
# Library
go get github.com/gemineo/pack2d

# CLI
go install github.com/gemineo/pack2d/cmd/pack2d@latest
```

## Library Usage

```go
import "github.com/gemineo/pack2d"

// Encode — compresses and base45-encodes a payload
encoded, stats, err := pack2d.Encode(
    []byte(`{"patient":"John","id":"12345"}`),
    pack2d.WithInputType(pack2d.JSON),
)

// Decode — header byte is self-describing; no options needed
decoded, stats, err := pack2d.Decode(encoded)

// Generate a QR code image
png, stats, err := pack2d.GenerateBarcode(
    []byte(`{"patient":"John","id":"12345"}`),
    pack2d.WithInputType(pack2d.JSON),
    pack2d.WithBarcodeType(pack2d.QRCode),
    pack2d.WithSize(512),
    pack2d.WithErrorCorrection(pack2d.ECHigh),
)
os.WriteFile("patient.png", png, 0o644)

// Inspect an encoded string
result, err := pack2d.Inspect(encoded)
fmt.Println(result.Compression, result.Serialization)
```

### Reusable Encoder/Decoder

```go
enc := pack2d.NewEncoder(
    pack2d.WithInputType(pack2d.JSON),
    pack2d.WithCompression(pack2d.Zlib),
)
for _, payload := range payloads {
    encoded, stats, err := enc.Encode(payload)
    // ...
}
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithInputType(t)` | `Raw` | `Raw`, `JSON` — serialization applied before compression |
| `WithEncoding(s)` | `"utf-8"` | Source charset hint (IANA name); BOM auto-detected |
| `WithCompression(c)` | `Zlib` | Compression algorithm |
| `WithCompressionLevel(n)` | algorithm default | Algorithm-specific level |
| `WithBarcodeType(t)` | `QRCode` | `QRCode`, `DataMatrix` |
| `WithImageFormat(f)` | `PNG` | `PNG`, `SVG` |
| `WithSize(n)` | `256` | Image dimension in pixels |
| `WithErrorCorrection(ec)` | `ECMedium` | QR only: `ECLow`, `ECMedium`, `ECQuarter`, `ECHigh` |
| `WithQuietZone(n)` | `4` | Quiet zone width in modules |

## CLI Usage

```bash
# Encode from stdin
echo '{"patient":"John","id":"12345"}' | pack2d encode -t json

# Decode
echo '<encoded>' | pack2d decode

# Generate a QR code
pack2d barcode -t json -b qrcode -f png -s 512 -o patient.png < patient.json

# Generate a DataMatrix SVG
pack2d barcode -t json -b datamatrix -f svg -o label.svg < data.json

# Inspect encoded metadata
echo '<encoded>' | pack2d inspect

# Unix pipe composition
cat patient.json | pack2d encode -t json -q | pack2d barcode --pre-encoded -b qrcode -o patient.png
```

### Subcommands

| Command | Description |
|---------|-------------|
| `encode` | Compress and base45-encode a payload |
| `decode` | Decode a base45 string back to original data |
| `barcode` | Encode and generate a barcode image in one step |
| `inspect` | Show header metadata and barcode feasibility |
| `version` | Print version information |

Stats are written to stderr; encoded/decoded data goes to stdout. Use `-q` / `--quiet` to suppress stats for piping.

Exit codes: `0` success, `1` user error, `2` system/I/O error.

## Sub-packages

All sub-packages are public and independently importable:

```go
import "github.com/gemineo/pack2d/compress"  // Compressor interface + zlib
import "github.com/gemineo/pack2d/barcode"   // Generator interface + QR + DataMatrix
import "github.com/gemineo/pack2d/codec"     // Header byte pack/unpack
import "github.com/gemineo/pack2d/dict"      // Dictionary Store interface + MemoryStore
import "github.com/gemineo/pack2d/encoding"  // Base45 encode/decode
import "github.com/gemineo/pack2d/serial"    // Serializer interface + raw + JSON
import "github.com/gemineo/pack2d/textenc"   // UTF-8 normalization
```

Each component (`Compressor`, `Serializer`, `Generator`, `dict.Store`) is an interface — custom implementations can be registered without modifying library code.

