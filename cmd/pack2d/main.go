// Command pack2d is the CLI for the pack2d library.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gemineo/pack2d"
	"github.com/gemineo/pack2d/dict"
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
		fmt.Fprintln(os.Stderr, "\nRun \"pack2d help\" for usage.")
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
	case "dict":
		runDict(os.Args[2:])
	case "version":
		runVersion(os.Args[2:])
	case "help", "-h", "--help":
		runHelp(os.Args[2:])
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
	fmt.Fprintln(w, "  encode   Compress and base45-encode a payload")
	fmt.Fprintln(w, "  decode   Decode a base45 string back to original data")
	fmt.Fprintln(w, "  barcode  Encode data and generate a barcode image")
	fmt.Fprintln(w, "  inspect  Show header metadata and barcode feasibility")
	fmt.Fprintln(w, "  dict     Manage compression dictionaries (list, train, bench)")
	fmt.Fprintln(w, "  version  Print version and platform information")
	fmt.Fprintln(w, "  help     Show this help or options for a specific command")
}

func runHelp(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		fmt.Fprintln(os.Stdout, "\nRun \"pack2d help <command>\" for options of a specific command.")
		return
	}
	switch args[0] {
	case "encode":
		printCommandHelp("encode", "Compress and base45-encode a payload.", func(fs *flag.FlagSet) {
			fs.String("input", "", "input file (default: stdin)")
			fs.String("t", "raw", "input type: raw, json, xml, cbor")
			fs.String("c", "zlib", "compression algorithm: zlib, zstd, brotli")
			fs.Int("l", -1, "compression level (-1 = algorithm default)")
			fs.Bool("q", false, "suppress stats output")
		})
	case "decode":
		printCommandHelp("decode", "Decode a base45 string back to original data.", func(fs *flag.FlagSet) {
			fs.String("input", "", "input file (default: stdin)")
			fs.Bool("q", false, "suppress stats output")
		})
	case "barcode":
		printCommandHelp("barcode", "Encode data and generate a barcode image.", func(fs *flag.FlagSet) {
			fs.String("input", "", "input file (default: stdin)")
			fs.String("t", "raw", "input type: raw, json, xml, cbor")
			fs.String("c", "zlib", "compression algorithm: zlib, zstd, brotli")
			fs.Int("l", -1, "compression level (-1 = algorithm default)")
			fs.String("b", "qrcode", "barcode type: qrcode, datamatrix")
			fs.String("f", "png", "image format: png, svg")
			fs.String("o", "", "output file (required)")
			fs.Int("s", 256, "image size in pixels")
			fs.Bool("q", false, "suppress stats output")
		})
	case "inspect":
		printCommandHelp("inspect", "Show header metadata and barcode feasibility.", func(fs *flag.FlagSet) {
			fs.String("input", "", "input file (default: stdin)")
		})
	case "dict":
		fmt.Fprintln(os.Stdout, "Usage: pack2d dict <subcommand> [options]")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Subcommands:")
		fmt.Fprintln(os.Stdout, "  list   List dictionaries in a directory")
		fmt.Fprintln(os.Stdout, "  train  Train a new dictionary from sample files (requires zstd in PATH)")
		fmt.Fprintln(os.Stdout, "  bench  Benchmark dictionary compression on sample files")
	case "version":
		printCommandHelp("version", "Print version and platform information.", nil)
	default:
		fmt.Fprintf(os.Stderr, "pack2d help: unknown command %q\n", args[0])
		os.Exit(1)
	}
}

func printCommandHelp(name, desc string, setup func(*flag.FlagSet)) {
	fmt.Fprintf(os.Stdout, "Usage: pack2d %s [options]\n\n%s\n", name, desc)
	if setup != nil {
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		setup(fs)
		fmt.Fprintln(os.Stdout, "\nOptions:")
		fs.PrintDefaults()
	}
}

func runEncode(args []string) {
	fs := flag.NewFlagSet("encode", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	inputType := fs.String("t", "raw", "input type: raw, json, xml, cbor")
	compression := fs.String("c", "zlib", "compression algorithm: zlib, zstd, brotli")
	level := fs.Int("l", -1, "compression level (-1 = algorithm default)")
	quiet := fs.Bool("q", false, "suppress stats output")
	_ = fs.Parse(args)

	data := readInput(*input, fs)

	opts := []pack2d.Option{
		pack2d.WithInputType(pack2d.InputType(*inputType)),
		pack2d.WithCompression(pack2d.CompressionType(*compression)),
		pack2d.WithCompressionLevel(*level),
	}

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

	if _, err := os.Stdout.Write(decoded); err != nil {
		fmt.Fprintf(os.Stderr, "pack2d decode: write stdout: %v\n", err)
		os.Exit(2)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "stats: encoded=%d chars, compressed=%d bytes, output=%d bytes\n",
			stats.EncodedBytes, stats.CompressedBytes, stats.InputBytes)
	}
}

func runBarcode(args []string) {
	fs := flag.NewFlagSet("barcode", flag.ExitOnError)
	input := fs.String("input", "", "input file (default: stdin)")
	inputType := fs.String("t", "raw", "input type: raw, json, xml, cbor")
	compression := fs.String("c", "zlib", "compression algorithm: zlib, zstd, brotli")
	level := fs.Int("l", -1, "compression level (-1 = algorithm default)")
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
		pack2d.WithCompression(pack2d.CompressionType(*compression)),
		pack2d.WithCompressionLevel(*level),
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

// runDict dispatches pack2d dict subcommands.
func runDict(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "pack2d dict: subcommand required (list, train, bench)")
		fmt.Fprintln(os.Stderr, "Run \"pack2d help dict\" for usage.")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		runDictList(args[1:])
	case "train":
		runDictTrain(args[1:])
	case "bench":
		runDictBench(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pack2d dict: unknown subcommand %q\n", args[0])
		os.Exit(1)
	}
}

func runDictList(args []string) {
	fs := flag.NewFlagSet("dict list", flag.ExitOnError)
	dir := fs.String("dir", ".", "directory containing dictionaries")
	_ = fs.Parse(args)

	store, err := dict.NewFilesystemStore(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict list: %v\n", err)
		os.Exit(1)
	}

	entries, err := store.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict list: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("no dictionaries found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tSAMPLES\tCREATED\tDESCRIPTION")
	for _, d := range entries {
		fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\t%s\n",
			d.ID, d.Name, len(d.Data), d.SampleCount,
			d.CreatedAt.Format(time.RFC3339), d.Description)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict list: flush: %v\n", err)
		os.Exit(2)
	}
}

func runDictTrain(args []string) {
	fs := flag.NewFlagSet("dict train", flag.ExitOnError)
	name := fs.String("name", "", "dictionary name (required)")
	samplesDir := fs.String("samples", "", "directory of sample files (required)")
	outputDir := fs.String("output", ".", "directory to save the dictionary")
	description := fs.String("description", "", "optional description")
	_ = fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "pack2d dict train: --name is required")
		os.Exit(1)
	}
	if *samplesDir == "" {
		fmt.Fprintln(os.Stderr, "pack2d dict train: --samples is required")
		os.Exit(1)
	}

	samples, err := loadSamplesFromDir(*samplesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict train: load samples: %v\n", err)
		os.Exit(1)
	}
	if len(samples) == 0 {
		fmt.Fprintln(os.Stderr, "pack2d dict train: no sample files found in directory")
		os.Exit(1)
	}

	dictData, err := dict.Train(samples, "zstd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict train: %v\n", err)
		fmt.Fprintln(os.Stderr, "note: dict train requires the 'zstd' binary in PATH")
		os.Exit(1)
	}

	store, err := dict.NewFilesystemStore(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict train: open store: %v\n", err)
		os.Exit(1)
	}

	entry := &dict.Dictionary{
		Name:        *name,
		Description: *description,
		Data:        dictData,
		CreatedAt:   time.Now().UTC(),
		SampleCount: len(samples),
	}
	if err := store.Save(entry); err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict train: save: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "trained dictionary %q (id=%d, size=%d bytes) from %d samples\n",
		entry.Name, entry.ID, len(entry.Data), len(samples))
}

func runDictBench(args []string) {
	fs := flag.NewFlagSet("dict bench", flag.ExitOnError)
	dictFile := fs.String("dict", "", "path to .dict file (required)")
	samplesDir := fs.String("samples", "", "directory of sample files (required)")
	_ = fs.Parse(args)

	if *dictFile == "" {
		fmt.Fprintln(os.Stderr, "pack2d dict bench: --dict is required")
		os.Exit(1)
	}
	if *samplesDir == "" {
		fmt.Fprintln(os.Stderr, "pack2d dict bench: --samples is required")
		os.Exit(1)
	}

	dictData, err := os.ReadFile(*dictFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict bench: read dict: %v\n", err)
		os.Exit(1)
	}

	samples, err := loadSamplesFromDir(*samplesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict bench: load samples: %v\n", err)
		os.Exit(1)
	}
	if len(samples) == 0 {
		fmt.Fprintln(os.Stderr, "pack2d dict bench: no sample files found")
		os.Exit(1)
	}

	results, err := dict.Benchmark(samples, dictData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict bench: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintf(w, "samples\t%d\n", results.SampleCount)
	fmt.Fprintf(w, "total input bytes\t%d\n", results.TotalInputBytes)
	fmt.Fprintf(w, "zstd (no dict) bytes\t%d\n", results.ZstdNoDictBytes)
	fmt.Fprintf(w, "zstd (with dict) bytes\t%d\n", results.ZstdWithDictBytes)
	fmt.Fprintf(w, "improvement\t%.1f%%\n", results.ImprovementPct)
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "pack2d dict bench: flush: %v\n", err)
		os.Exit(2)
	}
}

func loadSamplesFromDir(dir string) ([][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}
	var samples [][]byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read sample %q: %w", name, err)
		}
		samples = append(samples, data)
	}
	return samples, nil
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
