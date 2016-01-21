package msgpack

import (
	"io"

	"github.com/goadesign/goa"
	"github.com/ugorji/go/codec"
)

// Enforce that codec.Decoder satisfies goa.ResettableDecoder at compile time
var (
	_ goa.ResettableDecoder = (*codec.Decoder)(nil)
	_ goa.ResettableEncoder = (*codec.Encoder)(nil)
)

// Factory uses github.com/ugorji/go/codec to act as an DecoderFactory and EncoderFactory
type Factory struct{}

// create and configure Handle
var h codec.MsgpackHandle

// DecoderFactory is the default factory used by the goa `Consumes` DSL
func DecoderFactory() goa.EncoderFactory {
	return &Factory{}
}

// EncoderFactory is the default factory used by the goa `Produces` DSL
func EncoderFactory() goa.EncoderFactory {
	return &Factory{}
}

// NewDecoder returns a new json.Decoder that satisfies goa.Decoder
// The built in codec.Decoder has a compatible Reset() func
func (f *Factory) NewDecoder(r io.Reader) goa.Decoder {
	return codec.NewDecoder(r, &h)
}

// NewEncoder returns a new json.Encoder that satisfies goa.Encoder
// The built in codec.Encoder has a compatible Reset() func
func (f *Factory) NewEncoder(w io.Writer) goa.Encoder {
	return codec.NewEncoder(w, &h)
}
