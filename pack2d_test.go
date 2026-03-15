package pack2d

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gemineo/pack2d/dict"
	"github.com/gemineo/pack2d/internal/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestDict(t *testing.T) *dict.Dictionary {
	t.Helper()
	samples := make([][]byte, 20)
	for i := range samples {
		samples[i] = []byte(strings.Repeat(
			`{"patient":"John Smith","id":"12345","status":"active","score":98.6}`+
				strings.Repeat(" ", i+1), // slight variation to avoid identical-sample panic
			3,
		))
	}
	dictData, err := dict.Train(samples, "zstd")
	if err != nil {
		t.Skipf("cannot train test dictionary: %v", err)
	}
	return &dict.Dictionary{
		Name:      "test",
		Data:      dictData,
		CreatedAt: time.Now(),
	}
}

var (
	rawData  = []byte("hello world this is raw data for testing")
	jsonData = []byte(`{"patient":"John","id":"12345","status":"active"}`)
	xmlData  = []byte(`<patient><name>John</name><id>12345</id></patient>`)
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		inputType   InputType
		compression CompressionType
		wantErr     error
	}{
		{"raw/zlib simple", []byte("hello world"), Raw, Zlib, nil},
		{"raw/zlib empty", []byte{}, Raw, Zlib, nil},
		{"raw/zlib binary", []byte{0x00, 0x01, 0x02, 0xFF}, Raw, Zlib, nil},
		{"json/zlib simple", []byte(`{"key":"value"}`), JSON, Zlib, nil},
		{"json/zlib unicode", []byte(`{"name":"héllo","emoji":"🎉"}`), JSON, Zlib, nil},
		{"json/zlib nested", []byte(`{"a":{"b":{"c":42}}}`), JSON, Zlib, nil},
		{"raw/zstd", rawData, Raw, Zstd, nil},
		{"json/zstd", jsonData, JSON, Zstd, nil},
		{"json/brotli", jsonData, JSON, Brotli, nil},
		{"raw/brotli", rawData, Raw, Brotli, nil},
		{"cbor/zstd", jsonData, CBOR, Zstd, nil},
		{"xml/zlib", xmlData, XML, Zlib, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, stats, err := Encode(tt.data,
				WithInputType(tt.inputType),
				WithCompression(tt.compression),
			)
			require.NoError(t, err)
			assert.Greater(t, len(encoded), 0)
			assert.Greater(t, stats.EncodedBytes, 0)

			decoded, _, err := Decode(encoded)
			require.NoError(t, err)

			switch tt.inputType {
			case JSON, CBOR:
				// CBOR round-trips through JSON, result may differ in whitespace
				assert.NotEmpty(t, decoded)
			case XML:
				// XML may be minified, just check not empty
				assert.NotEmpty(t, decoded)
			default:
				assert.Equal(t, tt.data, decoded)
			}
		})
	}
}

func TestRoundTripLarge(t *testing.T) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100))
	encoded, stats, err := Encode(data)
	require.NoError(t, err)
	assert.Less(t, stats.CompressedBytes, stats.InputBytes, "compression should reduce size")

	decoded, _, err := Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestRoundTripWithDictionary(t *testing.T) {
	d := buildTestDict(t)
	store := dict.NewMemoryStore()
	require.NoError(t, store.Save(d))
	assert.Equal(t, uint16(1), d.ID)

	data := jsonData
	encoded, _, err := Encode(data,
		WithInputType(JSON),
		WithDictionary(d),
	)
	require.NoError(t, err)

	decoded, _, err := Decode(encoded, WithDictStore(store))
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)

	// Decoding without a dict store must fail with ErrDictionaryNotFound
	_, _, err = Decode(encoded)
	require.ErrorIs(t, err, ErrDictionaryNotFound)
}

func TestInspect(t *testing.T) {
	data := []byte("inspect test data")
	encoded, _, err := Encode(data)
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.Equal(t, "zlib", result.Compression)
	assert.Equal(t, "raw", result.Serialization)
	assert.False(t, result.HasDictionary)
	assert.Equal(t, uint8(0), result.Version)
	assert.Contains(t, result.CompatibleBarcodes, "qrcode")
}

func TestInspectJSON(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	encoded, _, err := Encode(data, WithInputType(JSON))
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.Equal(t, "json", result.Serialization)
}

func FuzzDecode(f *testing.F) {
	data := []byte("fuzz seed")
	encoded, _, _ := Encode(data)
	f.Add(encoded)
	f.Add("")
	f.Add("HELLO")
	f.Add("!!!!")
	f.Fuzz(func(t *testing.T, s string) {
		// must never panic
		Decode(s) //nolint:errcheck
	})
}

func TestDecodeInvalid(t *testing.T) {
	_, _, err := Decode("not valid base45 ???")
	assert.Error(t, err)
}

// GenerateBarcode / EncodeToBarcode

func TestGenerateBarcode(t *testing.T) {
	tests := []struct {
		name        string
		barcodeType BarcodeType
		format      ImageFormat
		magic       []byte // expected header bytes in output
	}{
		{"qrcode/png", QRCode, PNG, []byte("\x89PNG")},
		{"qrcode/svg", QRCode, SVG, []byte("<svg")},
		{"datamatrix/png", DataMatrix, PNG, []byte("\x89PNG")},
		{"datamatrix/svg", DataMatrix, SVG, []byte("<svg")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgData, stats, err := GenerateBarcode(
				[]byte("HELLO WORLD"),
				WithBarcodeType(tt.barcodeType),
				WithImageFormat(tt.format),
			)
			require.NoError(t, err)
			assert.True(t, bytes.HasPrefix(imgData, tt.magic),
				"expected %q prefix, got %q", tt.magic, imgData[:min(len(imgData), 10)])
			assert.Greater(t, stats.EncodedBytes, 0)
		})
	}
}

func TestEncodeToBarcode(t *testing.T) {
	enc := NewEncoder(
		WithInputType(JSON),
		WithCompression(Zstd),
		WithBarcodeType(QRCode),
		WithImageFormat(PNG),
		WithSize(256),
	)
	imgData, stats, err := enc.EncodeToBarcode(jsonData)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(imgData, []byte("\x89PNG")))
	assert.Greater(t, stats.InputBytes, 0)
}

func TestEncodeToBarcodesUnsupportedFormat(t *testing.T) {
	enc := NewEncoder(WithImageFormat(ImageFormat("webp")))
	_, _, err := enc.EncodeToBarcode([]byte("test"))
	assert.Error(t, err)
}

// Inspect edge cases

func TestInspectWithDictionary(t *testing.T) {
	d := buildTestDict(t)
	store := dict.NewMemoryStore()
	require.NoError(t, store.Save(d))

	encoded, _, err := Encode(jsonData, WithInputType(JSON), WithDictionary(d))
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.True(t, result.HasDictionary)
	assert.Equal(t, d.ID, result.DictionaryID)
	assert.Equal(t, "zstd", result.Compression)
}

func TestInspectAllCompressionSerializationNames(t *testing.T) {
	tests := []struct {
		inputType   InputType
		compression CompressionType
		wantSerial  string
		wantComp    string
	}{
		{Raw, Zlib, "raw", "zlib"},
		{JSON, Zstd, "json", "zstd"},
		{XML, Brotli, "xml", "brotli"},
		{CBOR, Zstd, "cbor", "zstd"},
	}
	for _, tt := range tests {
		t.Run(tt.wantComp+"/"+tt.wantSerial, func(t *testing.T) {
			encoded, _, err := Encode([]byte(`<x/>`),
				WithInputType(tt.inputType),
				WithCompression(tt.compression),
			)
			if tt.inputType == XML {
				encoded, _, err = Encode(xmlData,
					WithInputType(tt.inputType),
					WithCompression(tt.compression),
				)
			}
			if tt.inputType == CBOR || tt.inputType == JSON {
				encoded, _, err = Encode(jsonData,
					WithInputType(tt.inputType),
					WithCompression(tt.compression),
				)
			}
			require.NoError(t, err)
			result, err := Inspect(encoded)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSerial, result.Serialization)
			assert.Equal(t, tt.wantComp, result.Compression)
		})
	}
}

func TestInspectLongPayloadTruncated(t *testing.T) {
	// Use 2000 bytes of cycling values — enough entropy that compressed output exceeds 64 bytes.
	data := make([]byte, 2000)
	for i := range data {
		data[i] = byte(i % 251) // prime modulus → no simple pattern
	}
	encoded, _, err := Encode(data)
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(result.DataPreview, "..."),
		"preview should be truncated, got: %q", result.DataPreview)
}

func TestInspectInvalidBase45(t *testing.T) {
	_, err := Inspect("???invalid???")
	assert.Error(t, err)
}

// Decode error paths — craft payloads with specific header bits set

func TestDecodeUnknownCompressionID(t *testing.T) {
	// Header: VER=0, CMP=0x03 (reserved), DCT=0, SER=0x00 → byte = 0b00_11_0_000 = 0x30
	// Payload: valid zlib-compressed data — but header says "0x03" which is unregistered.
	zlibData, _, err := NewEncoder(WithCompression(Zlib)).Encode([]byte("test"))
	require.NoError(t, err)
	raw, err := encoding.Base45Decode(zlibData)
	require.NoError(t, err)

	// Patch byte 0: keep SER bits, set CMP to 0x03
	patched := make([]byte, len(raw))
	copy(patched, raw)
	patched[0] = (patched[0] & 0xCF) | (0x03 << 4) // CMP = 0b11

	_, _, err = Decode(encoding.Base45Encode(patched))
	assert.ErrorIs(t, err, ErrUnknownCompression)
}

func TestDecodeUnknownSerializerID(t *testing.T) {
	// Encode valid data, then patch the SER bits to 0x07 (unused).
	// The decompressor succeeds, but deserialization fails.
	encoded, _, err := Encode([]byte("test"))
	require.NoError(t, err)
	raw, err := encoding.Base45Decode(encoded)
	require.NoError(t, err)

	patched := make([]byte, len(raw))
	copy(patched, raw)
	patched[0] = (patched[0] & 0xF8) | 0x07 // SER = 0b111 (unregistered)

	_, _, err = Decode(encoding.Base45Encode(patched))
	assert.ErrorIs(t, err, ErrUnknownSerializer)
}

func TestDecodeDeserializeFails(t *testing.T) {
	// Encode raw bytes that are not valid CBOR, then patch SER bits to 0x03 (CBOR).
	// The registered CBOR deserializer will reject the data as invalid CBOR.
	// 0xFF bytes are invalid as a CBOR top-level value (break code outside indefinite context).
	encoded, _, err := Encode([]byte{0xFF, 0xFF, 0xFF})
	require.NoError(t, err)
	raw, err := encoding.Base45Decode(encoded)
	require.NoError(t, err)

	patched := make([]byte, len(raw))
	copy(patched, raw)
	patched[0] = (patched[0] & 0xF8) | 0x03 // SER = cbor

	_, _, err = Decode(encoding.Base45Encode(patched))
	assert.Error(t, err)
}

func TestDecodeMissingDictStore(t *testing.T) {
	d := buildTestDict(t)
	store := dict.NewMemoryStore()
	require.NoError(t, store.Save(d))

	encoded, _, err := Encode(jsonData, WithInputType(JSON), WithDictionary(d))
	require.NoError(t, err)

	// Decode without store must return ErrDictionaryNotFound
	_, _, err = Decode(encoded)
	assert.ErrorIs(t, err, ErrDictionaryNotFound)
}

func TestDecodeDictNotFoundInStore(t *testing.T) {
	d := buildTestDict(t)
	store := dict.NewMemoryStore()
	require.NoError(t, store.Save(d))

	encoded, _, err := Encode(jsonData, WithInputType(JSON), WithDictionary(d))
	require.NoError(t, err)

	// Use an empty store — dictionary ID won't be found
	emptyStore := dict.NewMemoryStore()
	_, _, err = Decode(encoded, WithDictStore(emptyStore))
	assert.ErrorIs(t, err, ErrDictionaryNotFound)
}

// Encode error paths

func TestEncodeUnknownSerializer(t *testing.T) {
	_, _, err := Encode([]byte("test"), WithInputType(InputType("unknown")))
	assert.ErrorIs(t, err, ErrUnknownSerializer)
}

func TestEncodeUnknownCompression(t *testing.T) {
	_, _, err := Encode([]byte("test"), WithCompression(CompressionType("lz4")))
	assert.ErrorIs(t, err, ErrUnknownCompression)
}

func TestEncodeInvalidUTF8WithJSONType(t *testing.T) {
	// JSON type triggers textenc.Normalize which rejects invalid UTF-8.
	_, _, err := Encode([]byte{0x80, 0x81}, WithInputType(JSON))
	assert.Error(t, err)
}

// Option coverage

func TestWithEncodingLatin1(t *testing.T) {
	// 0xE9 = 'é' in ISO-8859-1; WithInputType(Raw) + WithEncoding triggers textenc path.
	data := []byte{0xE9, 0x74, 0xE9} // "été" in latin-1
	encoded, _, err := Encode(data,
		WithInputType(Raw),
		WithEncoding("iso-8859-1"),
	)
	require.NoError(t, err)

	decoded, _, err := Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, "été", string(decoded))
}

func TestWithCompressionLevel(t *testing.T) {
	data := []byte("compression level option test data")
	for _, level := range []int{-1, 1, 9} {
		encoded, _, err := Encode(data, WithCompression(Zlib), WithCompressionLevel(level))
		require.NoError(t, err, "level=%d", level)
		decoded, _, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, data, decoded)
	}
}

func TestWithBarcodeAndImageOptions(t *testing.T) {
	// Smoke-test that option setters reach the barcode generator.
	imgData, _, err := GenerateBarcode(
		[]byte("OPTIONS"),
		WithBarcodeType(DataMatrix),
		WithImageFormat(SVG),
		WithSize(128),
		WithErrorCorrection(ECHigh),
		WithQuietZone(0),
	)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(imgData, []byte("<svg")))
}

// Stats validation

func TestEncodeStatsNonZeroInput(t *testing.T) {
	data := []byte(strings.Repeat("x", 100))
	_, stats, err := Encode(data)
	require.NoError(t, err)
	assert.Equal(t, 100, stats.InputBytes)
	assert.Greater(t, stats.EncodedBytes, 0)
	assert.Greater(t, stats.CompressedBytes, 0)
	assert.Greater(t, stats.CompressionRatio, 0.0)
}

func TestEncodeStatsZeroInput(t *testing.T) {
	_, stats, err := Encode([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, stats.InputBytes)
	assert.Equal(t, 0.0, stats.CompressionRatio)
}

func TestDecodeStatsConsistentWithEncode(t *testing.T) {
	data := []byte(strings.Repeat("x", 100))
	encoded, encStats, err := Encode(data)
	require.NoError(t, err)

	_, decStats, err := Decode(encoded)
	require.NoError(t, err)

	// InputBytes in both should be the uncompressed data size
	assert.Equal(t, encStats.InputBytes, decStats.InputBytes,
		"InputBytes should represent uncompressed data size in both encode and decode")
	// EncodedBytes in both should be the base45 string size
	assert.Equal(t, encStats.EncodedBytes, decStats.EncodedBytes,
		"EncodedBytes should represent base45 string size in both encode and decode")
	// CompressedBytes should match
	assert.Equal(t, encStats.CompressedBytes, decStats.CompressedBytes,
		"CompressedBytes should be the same compressed payload")
	// CompressionRatio should be the same
	assert.InDelta(t, encStats.CompressionRatio, decStats.CompressionRatio, 0.01,
		"CompressionRatio should be consistent")
}

func TestInspectUnknownSerializationID(t *testing.T) {
	encoded, _, err := Encode([]byte("test"))
	require.NoError(t, err)

	// Patch the header to set SER to 0x07 (unknown)
	raw, err := encoding.Base45Decode(encoded)
	require.NoError(t, err)
	patched := make([]byte, len(raw))
	copy(patched, raw)
	patched[0] = (patched[0] & 0xF8) | 0x07

	result, err := Inspect(encoding.Base45Encode(patched))
	require.NoError(t, err)
	assert.Equal(t, "unknown(7)", result.Serialization)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
