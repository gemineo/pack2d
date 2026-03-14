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
    pack2d.WithCompression(pack2d.Zstd),
)

// Decode — header byte is self-describing; no options needed for non-dict payloads
decoded, stats, err := pack2d.Decode(encoded)

// Generate a QR code image
png, stats, err := pack2d.GenerateBarcode(
    []byte(`{"patient":"John","id":"12345"}`),
    pack2d.WithInputType(pack2d.JSON),
    pack2d.WithCompression(pack2d.Brotli),
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
    pack2d.WithCompression(pack2d.Zstd),
)
for _, payload := range payloads {
    encoded, stats, err := enc.Encode(payload)
    // ...
}
```

### Dictionary-assisted compression

For domain-specific repetitive payloads (e.g. healthcare FHIR records), a pre-trained zstd dictionary can reduce output size significantly further.

```go
import "github.com/gemineo/pack2d/dict"

// Train a dictionary from representative sample data
samples := loadYourSamples() // [][]byte
dictData, err := dict.Train(samples, "zstd")

// Store in memory (or use dict.NewFilesystemStore for persistence)
store := dict.NewMemoryStore()
d := &dict.Dictionary{Name: "fhir-patient", Data: dictData}
store.Save(d) // auto-assigns ID

// Encode with dictionary
encoded, _, err := pack2d.Encode(payload,
    pack2d.WithInputType(pack2d.CBOR),
    pack2d.WithDictionary(d),
)

// Decode — must supply the same store
decoded, _, err := pack2d.Decode(encoded, pack2d.WithDictStore(store))
```

### Pipeline combinations

| Use case | InputType | Compression | Notes |
|----------|-----------|-------------|-------|
| Generic text | `Raw` | `Zlib` | Fastest, widest compat |
| JSON payload | `JSON` | `Zstd` | Fast + good ratio |
| JSON payload | `JSON` | `Brotli` | Best ratio, slower |
| Structured JSON | `CBOR` | `Zstd` | CBOR is 20–40% smaller than minified JSON before compression |
| Domain-specific | `CBOR` | `Zstd` + dictionary | Optimal for repetitive schemas (FHIR, EDI, etc.) |
| XML payload | `XML` | `Zstd` | Minifies XML whitespace before compression |

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithInputType(t)` | `Raw` | `Raw`, `JSON`, `XML`, `CBOR` — serialization applied before compression |
| `WithEncoding(s)` | `"utf-8"` | Source charset hint (IANA name); BOM auto-detected |
| `WithCompression(c)` | `Zlib` | `Zlib`, `Zstd`, `Brotli` |
| `WithCompressionLevel(n)` | algorithm default | zlib: 0–9; zstd: 1–4; brotli: 0–11; -1 = default |
| `WithDictionary(d)` | nil | Pre-trained zstd dictionary; activates DCT header bit |
| `WithDictStore(s)` | nil | `dict.Store` used to resolve dictionaries on decode |
| `WithBarcodeType(t)` | `QRCode` | `QRCode`, `DataMatrix` |
| `WithImageFormat(f)` | `PNG` | `PNG`, `SVG` |
| `WithSize(n)` | `256` | Image dimension in pixels |
| `WithErrorCorrection(ec)` | `ECMedium` | QR only: `ECLow`, `ECMedium`, `ECQuarter`, `ECHigh` |
| `WithQuietZone(n)` | `4` | Quiet zone width in modules |

## CLI Usage

```bash
# Encode from stdin (default: raw/zlib)
echo '{"patient":"John","id":"12345"}' | pack2d encode -t json

# Encode with explicit compression algorithm and level
echo '{"patient":"John","id":"12345"}' | pack2d encode -t json -c zstd
echo '{"patient":"John","id":"12345"}' | pack2d encode -t cbor -c brotli -l 9

# Decode (self-describing header; no flags needed)
echo '<encoded>' | pack2d decode

# Generate a QR code
pack2d barcode -t json -c zstd -b qrcode -f png -s 512 -o patient.png < patient.json

# Generate a DataMatrix SVG
pack2d barcode -t json -b datamatrix -f svg -o label.svg < data.json

# Inspect encoded metadata
echo '<encoded>' | pack2d inspect

# Dictionary management
pack2d dict list --dir /path/to/dicts
pack2d dict train --name fhir-patient --samples ./samples/ --output ./dicts/ --description "FHIR R4 patient bundle"
pack2d dict bench --dict ./dicts/0001_fhir-patient.dict --samples ./samples/
```

### Subcommands

| Command | Description |
|---------|-------------|
| `encode` | Compress and base45-encode a payload |
| `decode` | Decode a base45 string back to original data |
| `barcode` | Encode and generate a barcode image in one step |
| `inspect` | Show header metadata and barcode feasibility |
| `dict` | Manage compression dictionaries (`list`, `train`, `bench`) |
| `version` | Print version information |
| `help` | Show command list or options for a specific command |

### encode / barcode flags

| Flag | Default | Description |
|------|---------|-------------|
| `-t` | `raw` | Input type: `raw`, `json`, `xml`, `cbor` |
| `-c` | `zlib` | Compression: `zlib`, `zstd`, `brotli` |
| `-l` | `-1` | Compression level (-1 = algorithm default) |
| `-q` | false | Suppress stats output |
| `-b` | `qrcode` | Barcode type (barcode only): `qrcode`, `datamatrix` |
| `-f` | `png` | Image format (barcode only): `png`, `svg` |
| `-o` | — | Output file (barcode only, required) |
| `-s` | `256` | Image size in pixels (barcode only) |

Stats are written to stderr; encoded/decoded data goes to stdout. Use `-q` to suppress stats for piping.

Exit codes: `0` success, `1` user error, `2` system/I/O error.

`pack2d version` prints the binary version and OS/arch, e.g. `pack2d v0.2.0 (linux/amd64)`. When built from source without `-ldflags`, the version is derived from the module VCS metadata.

## Extension points

The `dict` package is publicly importable for consumers that need a custom dictionary store:

```go
import "github.com/gemineo/pack2d/dict"

// In-memory store (suitable for testing or single-process use)
store := dict.NewMemoryStore()

// Filesystem store (persistent, file naming: {id:04d}_{name}.dict + .json metadata)
store, err := dict.NewFilesystemStore("/path/to/dicts")

// Train a dictionary from sample data (uses zstd.BuildDict internally)
dictData, err := dict.Train(samples, "zstd")

// Benchmark dictionary effectiveness
result, err := dict.Benchmark(samples, dictData)
fmt.Printf("improvement: %.1f%%\n", result.ImprovementPct)
```

Implement `dict.Store` and pass it via `WithDictStore(store)` to plug in a custom backend (e.g. database, object storage).

All other sub-packages (`codec`, `compress`, `encoding`, `serial`, `barcode`, `textenc`) are under `internal/` and are not part of the public API.
