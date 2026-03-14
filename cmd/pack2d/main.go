// Command pack2d is the CLI for the pack2d library.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/gemineo/pack2d"
)

var version = "dev" // overridden by -ldflags "-X main.version=..."

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var commit, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				commit = s.Value[:7]
			} else {
				commit = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if commit != "" {
		return "dev-" + commit + dirty
	}
	return "dev"
}

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "encode":
		runEncode(os.Args[2:])
	case "decode":
		runDecode(os.Args[2:])
	case "barcode":
		runBarcode(os.Args[2:])
	case "inspect":
		runInspect(os.Args[2:])
	case "version":
		runVersion(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "pack2d: unknown command %q\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(1)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: pack2d <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  encode   Encode input data to a pack2d base45 string")
	fmt.Fprintln(w, "  decode   Decode a pack2d base45 string to original data")
	fmt.Fprintln(w, "  barcode  Encode data and generate a barcode image")
	fmt.Fprintln(w, "  inspect  Inspect a pack2d encoded string metadata")
	fmt.Fprintln(w, "  version  Print version information")
}

func runEncode(args []string) {
	fs := flag.NewFlagSet("encode", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	inputType := fs.String("t", "raw", "input type: raw, json")
	quiet := fs.Bool("q", false, "suppress stats output")
	_ = fs.Parse(args)

	data := readInput(*input, fs)

	var opts []pack2d.Option
	opts = append(opts, pack2d.WithInputType(pack2d.InputType(*inputType)))

	encoded, stats, err := pack2d.Encode(data, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d encode: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(encoded)

	if !*quiet {
		fmt.Fprintf(os.Stderr, "stats: input=%d bytes, compressed=%d bytes, encoded=%d chars, ratio=%.2f\n",
			stats.InputBytes, stats.CompressedBytes, stats.EncodedBytes, stats.CompressionRatio)
	}
}

func runDecode(args []string) {
	fs := flag.NewFlagSet("decode", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	quiet := fs.Bool("q", false, "suppress stats output")
	_ = fs.Parse(args)

	data := readInput(*input, fs)
	// trim trailing newline from stdin
	encoded := string(data)
	if len(encoded) > 0 && encoded[len(encoded)-1] == '\n' {
		encoded = encoded[:len(encoded)-1]
	}

	decoded, stats, err := pack2d.Decode(encoded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d decode: %v\n", err)
		os.Exit(2)
	}

	os.Stdout.Write(decoded)

	if !*quiet {
		fmt.Fprintf(os.Stderr, "stats: encoded=%d chars, compressed=%d bytes, output=%d bytes\n",
			stats.InputBytes, stats.CompressedBytes, len(decoded))
	}
}

func runBarcode(args []string) {
	fs := flag.NewFlagSet("barcode", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	inputType := fs.String("t", "raw", "input type: raw, json")
	barcodeType := fs.String("b", "qrcode", "barcode type: qrcode, datamatrix")
	format := fs.String("f", "png", "image format: png, svg")
	output := fs.String("o", "", "output file (required)")
	size := fs.Int("s", 256, "image size in pixels")
	quiet := fs.Bool("q", false, "suppress stats output")
	_ = fs.Parse(args)

	if *output == "" {
		fmt.Fprintln(os.Stderr, "pack2d barcode: -o output file is required")
		os.Exit(1)
	}

	data := readInput(*input, fs)

	opts := []pack2d.Option{
		pack2d.WithInputType(pack2d.InputType(*inputType)),
		pack2d.WithBarcodeType(pack2d.BarcodeType(*barcodeType)),
		pack2d.WithImageFormat(pack2d.ImageFormat(*format)),
		pack2d.WithSize(*size),
	}

	imgData, stats, err := pack2d.GenerateBarcode(data, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d barcode: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, imgData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "pack2d barcode: write file: %v\n", err)
		os.Exit(2)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "stats: input=%d bytes, compressed=%d bytes, encoded=%d chars\n",
			stats.InputBytes, stats.CompressedBytes, stats.EncodedBytes)
		fmt.Fprintf(os.Stderr, "barcode: written to %s\n", *output)
	}
}

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	_ = fs.Parse(args)

	data := readInput(*input, fs)
	encoded := string(data)
	if len(encoded) > 0 && encoded[len(encoded)-1] == '\n' {
		encoded = encoded[:len(encoded)-1]
	}

	result, err := pack2d.Inspect(encoded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d inspect: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("version:       %d\n", result.Version)
	fmt.Printf("compression:   %s\n", result.Compression)
	fmt.Printf("serialization: %s\n", result.Serialization)
	fmt.Printf("dictionary:    %v\n", result.HasDictionary)
	if result.HasDictionary {
		fmt.Printf("dictionary-id: %d\n", result.DictionaryID)
	}
	fmt.Printf("barcodes:      %v\n", result.CompatibleBarcodes)
	fmt.Printf("header-byte:   %s\n", result.Header)
}

func runVersion(_ []string) {
	fmt.Printf("pack2d %s (%s/%s)\n", resolveVersion(), runtime.GOOS, runtime.GOARCH)
}

func readInput(path string, fs *flag.FlagSet) []byte {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pack2d: read file %q: %v\n", path, err)
			fs.Usage()
			os.Exit(2)
		}
		return data
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d: read stdin: %v\n", err)
		os.Exit(2)
	}
	return data
}
