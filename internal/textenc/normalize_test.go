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

func TestDetect(t *testing.T) {
	enc, err := Detect([]byte("valid utf-8"))
	require.NoError(t, err)
	assert.Equal(t, "utf-8", enc)

	enc, err = Detect([]byte{0x80, 0x81})
	require.NoError(t, err)
	assert.Equal(t, "unknown", enc)
}
