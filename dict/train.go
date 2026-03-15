package dict

import (
	"fmt"

	kdict "github.com/klauspost/compress/dict"
	"github.com/klauspost/compress/zstd"
)

// BenchmarkResult holds the results of a dictionary compression benchmark.
type BenchmarkResult struct {
	SampleCount       int
	TotalInputBytes   int
	ZstdNoDictBytes   int
	ZstdWithDictBytes int
	ImprovementPct    float64
}

// Train creates a zstd dictionary from the provided sample data.
// algo must be "zstd".
// Returns the raw dictionary bytes (valid zstd dictionary format with magic number 0xEC30A437).
func Train(samples [][]byte, algo string) ([]byte, error) {
	if algo != "zstd" {
		return nil, fmt.Errorf("dict train: unsupported algorithm %q (only zstd is supported)", algo)
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("dict train: no samples provided")
	}

	dictBytes, err := kdict.BuildZstdDict(samples, kdict.Options{
		MaxDictSize: 1 << 14, // 16 KiB
		HashBytes:   4,
	})
	if err != nil {
		return nil, fmt.Errorf("dict train: build dict: %w", err)
	}
	return dictBytes, nil
}

// Benchmark measures zstd compression with and without a dictionary across samples.
func Benchmark(samples [][]byte, dictionary []byte) (*BenchmarkResult, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("dict benchmark: no samples provided")
	}

	encNoDict, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, fmt.Errorf("dict benchmark: create encoder (no dict): %w", err)
	}
	defer encNoDict.Close()

	encWithDict, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(dictionary),
	)
	if err != nil {
		return nil, fmt.Errorf("dict benchmark: create encoder (with dict): %w", err)
	}
	defer encWithDict.Close()

	r := &BenchmarkResult{SampleCount: len(samples)}
	for _, s := range samples {
		r.TotalInputBytes += len(s)
		r.ZstdNoDictBytes += len(encNoDict.EncodeAll(s, nil))
		r.ZstdWithDictBytes += len(encWithDict.EncodeAll(s, nil))
	}

	if r.ZstdNoDictBytes > 0 {
		r.ImprovementPct = (1 - float64(r.ZstdWithDictBytes)/float64(r.ZstdNoDictBytes)) * 100
	}
	return r, nil
}
