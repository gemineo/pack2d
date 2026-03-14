package dict

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeSamples returns n samples each containing varied JSON-like content.
// zstd.BuildDict requires enough total data and cross-sample patterns; each sample is ~500 bytes.
func makeSamples(n int) [][]byte {
	samples := make([][]byte, n)
	fields := []string{
		`"status":"active"`, `"status":"inactive"`, `"status":"pending"`,
		`"type":"patient"`, `"type":"provider"`, `"type":"admin"`,
		`"department":"cardiology"`, `"department":"neurology"`, `"department":"oncology"`,
	}
	for i := range samples {
		base := fmt.Sprintf(
			`{"id":%d,"name":"Subject %d","score":%d,%s,"tags":["a","b","c"],"nested":{"x":%d,"y":%d}}`,
			i, i, i%100, fields[i%len(fields)], i*2, i*3,
		)
		samples[i] = []byte(strings.Repeat(base, 6))
	}
	return samples
}

func TestTrainBasic(t *testing.T) {
	samples := makeSamples(30)
	dictData, err := Train(samples, "zstd")
	require.NoError(t, err)
	assert.NotEmpty(t, dictData)
}

func TestTrainOutputIsValidZstdDict(t *testing.T) {
	// A valid zstd dictionary starts with magic number 0xEC30A437 (little-endian).
	samples := makeSamples(30)
	dictData, err := Train(samples, "zstd")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(dictData), 4)
	assert.Equal(t, byte(0x37), dictData[0])
	assert.Equal(t, byte(0xA4), dictData[1])
	assert.Equal(t, byte(0x30), dictData[2])
	assert.Equal(t, byte(0xEC), dictData[3])
}

func TestTrainEmptySamples(t *testing.T) {
	_, err := Train(nil, "zstd")
	assert.Error(t, err)

	_, err = Train([][]byte{}, "zstd")
	assert.Error(t, err)
}

func TestTrainUnsupportedAlgo(t *testing.T) {
	_, err := Train(makeSamples(5), "brotli")
	assert.Error(t, err)
}

func TestBenchmarkBasic(t *testing.T) {
	samples := makeSamples(30)
	dictData, err := Train(samples, "zstd")
	require.NoError(t, err)

	result, err := Benchmark(samples, dictData)
	require.NoError(t, err)
	assert.Equal(t, len(samples), result.SampleCount)
	assert.Greater(t, result.TotalInputBytes, 0)
	assert.Greater(t, result.ZstdNoDictBytes, 0)
	assert.Greater(t, result.ZstdWithDictBytes, 0)
}

func TestBenchmarkDictionaryImproves(t *testing.T) {
	// Highly repetitive samples — a trained dictionary should improve compression.
	samples := makeSamples(30)
	dictData, err := Train(samples, "zstd")
	require.NoError(t, err)

	result, err := Benchmark(samples, dictData)
	require.NoError(t, err)
	// Dictionary-aided should not be worse for repetitive data.
	assert.LessOrEqual(t, result.ZstdWithDictBytes, result.ZstdNoDictBytes)
}

func TestBenchmarkEmptySamples(t *testing.T) {
	_, err := Benchmark(nil, []byte("dict"))
	assert.Error(t, err)

	_, err = Benchmark([][]byte{}, []byte("dict"))
	assert.Error(t, err)
}
