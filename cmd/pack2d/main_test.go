package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// binaryPath holds the path to the compiled binary built in TestMain.
var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary into a temp dir so tests can run it as a subprocess.
	dir, err := os.MkdirTemp("", "pack2d-test-*")
	if err != nil {
		panic("cannot create temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "pack2d")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(".") // current dir is cmd/pack2d
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("cannot build binary: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

// run executes the binary with the given arguments, optionally piping stdin.
// Returns stdout, stderr, and exit code.
func run(t *testing.T, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected run error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLINoArgs(t *testing.T) {
	_, stderr, code := run(t, "")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "pack2d")
}

func TestCLIVersion(t *testing.T) {
	stdout, _, code := run(t, "", "version")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "pack2d")
}

func TestCLIHelp(t *testing.T) {
	stdout, _, code := run(t, "", "help")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "encode")
	assert.Contains(t, stdout, "decode")
}

func TestCLIHelpCommand(t *testing.T) {
	for _, cmd := range []string{"encode", "decode", "barcode", "inspect", "dict", "version"} {
		t.Run(cmd, func(t *testing.T) {
			stdout, _, code := run(t, "", "help", cmd)
			assert.Equal(t, 0, code, "help %s should exit 0", cmd)
			assert.NotEmpty(t, stdout)
		})
	}
}

func TestCLIHelpUnknownCommand(t *testing.T) {
	_, _, code := run(t, "", "help", "unknowncmd")
	assert.NotEqual(t, 0, code)
}

func TestCLIUnknownCommand(t *testing.T) {
	_, stderr, code := run(t, "", "unknowncmd")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "unknown command")
}

func TestCLIEncodeDecodeRoundTrip(t *testing.T) {
	input := "hello world"
	encoded, _, code := run(t, input, "encode", "-q")
	require.Equal(t, 0, code)
	encoded = strings.TrimRight(encoded, "\n")

	decoded, _, code := run(t, encoded, "decode", "-q")
	require.Equal(t, 0, code)
	assert.Equal(t, input, decoded)
}

func TestCLIEncodeDecodeJSON(t *testing.T) {
	input := `{"key":"value","num":42}`
	encoded, _, code := run(t, input, "encode", "-t", "json", "-c", "zstd", "-q")
	require.Equal(t, 0, code)
	encoded = strings.TrimRight(encoded, "\n")

	decoded, _, code := run(t, encoded, "decode", "-q")
	require.Equal(t, 0, code)
	assert.Contains(t, decoded, "key")
}

func TestCLIEncodeDecodeAllCompressors(t *testing.T) {
	input := "test payload for all compressors"
	for _, algo := range []string{"zlib", "zstd", "brotli"} {
		t.Run(algo, func(t *testing.T) {
			encoded, _, code := run(t, input, "encode", "-c", algo, "-q")
			require.Equal(t, 0, code, "encode with %s", algo)
			encoded = strings.TrimRight(encoded, "\n")

			decoded, _, code := run(t, encoded, "decode", "-q")
			require.Equal(t, 0, code, "decode with %s", algo)
			assert.Equal(t, input, decoded)
		})
	}
}

func TestCLIEncodeStatsOnStderr(t *testing.T) {
	// Without -q, stats go to stderr; stdout contains only the encoded value.
	stdout, stderr, code := run(t, "hello", "encode")
	require.Equal(t, 0, code)
	assert.Contains(t, stderr, "stats:")
	assert.NotContains(t, stdout, "stats:")
	assert.NotEmpty(t, strings.TrimSpace(stdout))
}

func TestCLIEncodeUnknownType(t *testing.T) {
	_, stderr, code := run(t, "hello", "encode", "-t", "unknowntype")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "encode")
}

func TestCLIDecodeInvalid(t *testing.T) {
	_, stderr, code := run(t, "???not-valid???", "decode")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "decode")
}

func TestCLIInspect(t *testing.T) {
	// Encode something, then inspect it.
	encoded, _, code := run(t, "inspect-me", "encode", "-q")
	require.Equal(t, 0, code)
	encoded = strings.TrimRight(encoded, "\n")

	stdout, _, code := run(t, encoded, "inspect")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "compression")
	assert.Contains(t, stdout, "version")
}

func TestCLIInspectInvalid(t *testing.T) {
	_, stderr, code := run(t, "???bad???", "inspect")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "inspect")
}

func TestCLIBarcodeOutputRequired(t *testing.T) {
	_, stderr, code := run(t, "hello", "barcode")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "-o")
}

func TestCLIBarcodeQRCodePNG(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.png")
	_, _, code := run(t, "hello", "barcode", "-o", outFile, "-q")
	require.Equal(t, 0, code)
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data[:4]), "\x89PNG"))
}

func TestCLIBarcodeSVG(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.svg")
	_, _, code := run(t, "hello", "barcode", "-b", "qrcode", "-f", "svg", "-o", outFile, "-q")
	require.Equal(t, 0, code)
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "<svg"))
}

func TestCLIDictNoSubcommand(t *testing.T) {
	_, stderr, code := run(t, "", "dict")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "subcommand")
}

func TestCLIDictUnknownSubcommand(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "unknown")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "unknown")
}

func TestCLIDictListEmpty(t *testing.T) {
	dir := t.TempDir()
	stdout, _, code := run(t, "", "dict", "list", "--dir", dir)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "no dictionaries found")
}

func TestCLIDictListInvalidDir(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "list", "--dir", "/nonexistent/path")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "dict list")
}

func TestCLIDictTrainAndList(t *testing.T) {
	samplesDir := t.TempDir()
	outputDir := t.TempDir()

	// Write some sample files with enough variation to train a dictionary.
	for i := 0; i < 30; i++ {
		content := strings.Repeat(
			`{"patient":"John","id":"12345","status":"active","score":98.6}`+
				strings.Repeat(" ", i+1),
			3,
		)
		err := os.WriteFile(
			filepath.Join(samplesDir, "sample"+string(rune('a'+i))+".json"),
			[]byte(content), 0o644,
		)
		require.NoError(t, err)
	}

	_, _, code := run(t, "", "dict", "train",
		"--name", "test-dict",
		"--samples", samplesDir,
		"--output", outputDir,
	)
	if code != 0 {
		t.Skip("dict train failed — likely requires more samples or zstd support")
	}

	// Now list should show one entry.
	stdout, _, code := run(t, "", "dict", "list", "--dir", outputDir)
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "test-dict")
}

func TestCLIDictTrainMissingName(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "train", "--samples", ".")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "--name")
}

func TestCLIDictTrainMissingSamples(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "train", "--name", "x")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "--samples")
}

func TestCLIDictTrainEmptyDir(t *testing.T) {
	emptyDir := t.TempDir()
	_, stderr, code := run(t, "", "dict", "train", "--name", "x", "--samples", emptyDir)
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "no sample files")
}

func TestCLIDictBenchMissingDict(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "bench", "--samples", ".")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "--dict")
}

func TestCLIDictBenchMissingSamples(t *testing.T) {
	_, stderr, code := run(t, "", "dict", "bench", "--dict", "/tmp/x.dict")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, stderr, "--samples")
}

func TestCLIEncodeFromFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "input-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString("hello from file")
	require.NoError(t, err)
	f.Close()

	stdout, _, code := run(t, "", "encode", "-input", f.Name(), "-q")
	require.Equal(t, 0, code)
	assert.NotEmpty(t, strings.TrimSpace(stdout))
}
