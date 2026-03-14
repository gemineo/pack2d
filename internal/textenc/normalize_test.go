package textenc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUTF8(t *testing.T) {
	data := []byte("hello world — Unicode: héllo")
	out, err := Normalize(data, "")
	require.NoError(t, err)
	assert.Equal(t, data, out)

	out, err = Normalize(data, "utf-8")
	require.NoError(t, err)
	assert.Equal(t, data, out)
}

func TestNormalizeInvalidUTF8(t *testing.T) {
	bad := []byte{0x80, 0x81, 0x82} // invalid UTF-8
	_, err := Normalize(bad, "")
	assert.Error(t, err)
}

func TestNormalizeUnknownEncoding(t *testing.T) {
	_, err := Normalize([]byte("data"), "not-a-real-encoding-xyz")
	assert.Error(t, err)
}

func TestNormalizeUTF16LEWithBOM(t *testing.T) {
	// "hi" encoded as UTF-16LE with BOM: FF FE 68 00 69 00
	// The x/text UTF-16LE decoder preserves the BOM as U+FEFF in output.
	data := []byte{0xFF, 0xFE, 0x68, 0x00, 0x69, 0x00}
	out, err := Normalize(data, "utf-8") // hint overridden by BOM
	require.NoError(t, err)
	assert.Contains(t, string(out), "hi")
	assert.True(t, len(out) > 0)
}

func TestNormalizeUTF16BEWithBOM(t *testing.T) {
	// "hi" encoded as UTF-16BE with BOM: FE FF 00 68 00 69
	// The x/text UTF-16BE decoder preserves the BOM as U+FEFF in output.
	data := []byte{0xFE, 0xFF, 0x00, 0x68, 0x00, 0x69}
	out, err := Normalize(data, "utf-8") // hint overridden by BOM
	require.NoError(t, err)
	assert.Contains(t, string(out), "hi")
	assert.True(t, len(out) > 0)
}

func TestNormalizeBOMWithShortData(t *testing.T) {
	// Single byte — no BOM detection possible, falls through to UTF-8 validation.
	data := []byte{0x41} // 'A'
	out, err := Normalize(data, "")
	require.NoError(t, err)
	assert.Equal(t, []byte{0x41}, out)
}

func TestNormalizeLatin1Transcoding(t *testing.T) {
	// 0xE9 = 'é' in ISO-8859-1; should transcode to UTF-8 as 0xC3 0xA9
	data := []byte{0xE9, 0x74, 0xE9} // "été" in latin-1
	out, err := Normalize(data, "iso-8859-1")
	require.NoError(t, err)
	assert.Equal(t, "été", string(out))
}

func TestNormalizeTrimSpacesHint(t *testing.T) {
	// Leading/trailing spaces in hint are tolerated.
	data := []byte("hello")
	out, err := Normalize(data, "  UTF-8  ")
	require.NoError(t, err)
	assert.Equal(t, data, out)
}

func TestDetect(t *testing.T) {
	enc, err := Detect([]byte("valid utf-8"))
	require.NoError(t, err)
	assert.Equal(t, "utf-8", enc)

	enc, err = Detect([]byte{0x80, 0x81})
	require.NoError(t, err)
	assert.Equal(t, "unknown", enc)
}
