package pack2d

import "github.com/gemineo/pack2d/dict"

type config struct {
	inputType        InputType
	encoding         string
	compression      CompressionType
	compressionLevel int
	dictionary       *dict.Dictionary
	barcodeType      BarcodeType
	imageFormat      ImageFormat
	size             int
	errorCorrection  ECLevel
	quietZone        int
	preEncoded       bool
	dictStore        dict.Store
}

func defaultConfig() config {
	return config{
		inputType:        Raw,
		encoding:         "utf-8",
		compression:      Zlib,
		compressionLevel: -1,
		barcodeType:      QRCode,
		imageFormat:      PNG,
		size:             256,
		errorCorrection:  ECMedium,
		quietZone:        4,
	}
}

// Option is a functional option for configuring Encoder and Decoder.
type Option func(*config)

// WithInputType sets the serialization type for the input.
func WithInputType(t InputType) Option {
	return func(c *config) { c.inputType = t }
}

// WithEncoding sets the text encoding hint for the input (e.g. "utf-8", "latin-1").
func WithEncoding(enc string) Option {
	return func(c *config) { c.encoding = enc }
}

// WithCompression sets the compression algorithm.
func WithCompression(comp CompressionType) Option {
	return func(c *config) { c.compression = comp }
}

// WithCompressionLevel sets the compression level (algorithm-specific).
func WithCompressionLevel(level int) Option {
	return func(c *config) { c.compressionLevel = level }
}

// WithBarcodeType sets the barcode symbology.
func WithBarcodeType(bt BarcodeType) Option {
	return func(c *config) { c.barcodeType = bt }
}

// WithImageFormat sets the output image format for barcode generation.
func WithImageFormat(f ImageFormat) Option {
	return func(c *config) { c.imageFormat = f }
}

// WithSize sets the barcode image size in pixels.
func WithSize(size int) Option {
	return func(c *config) { c.size = size }
}

// WithErrorCorrection sets the error-correction level for QR codes.
func WithErrorCorrection(ec ECLevel) Option {
	return func(c *config) { c.errorCorrection = ec }
}

// WithQuietZone sets the quiet zone size in modules (0 = no quiet zone).
func WithQuietZone(qz int) Option {
	return func(c *config) { c.quietZone = qz }
}

// WithDictStore sets the dictionary store used for dictionary-encoded payloads.
func WithDictStore(s dict.Store) Option {
	return func(c *config) { c.dictStore = s }
}
