package serial

type rawSerializer struct{}

// NewRawSerializer returns a pass-through Serializer (ID=0x00).
func NewRawSerializer() Serializer { return &rawSerializer{} }

func (s *rawSerializer) ID() byte     { return 0x00 }
func (s *rawSerializer) Name() string { return "raw" }

func (s *rawSerializer) Serialize(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func (s *rawSerializer) Deserialize(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
