package goaencoding

import (
	"io"

	"github.com/raphael/goa"
	"github.com/ugorji/go/codec"
)

type (
	// JSONFactory uses github.com/ugorji/go/codec to act as an DecoderFactory and EncoderFactory
	JSONFactory struct{}

	// MsgPackFactory uses github.com/ugorji/go/codec to act as an DecoderFactory and EncoderFactory
	MsgPackFactory struct{}

	// BincFactory uses github.com/ugorji/go/codec to act as an DecoderFactory and EncoderFactory
	BincFactory struct{}

	// CborFactory uses github.com/ugorji/go/codec to act as an DecoderFactory and EncoderFactory
	CborFactory struct{}
)

// create and configure Handle
var (
	jh codec.JsonHandle
	mh codec.MsgpackHandle
	bh codec.BincHandle
	ch codec.CborHandle

	// JSONDecoder is the default factory used by the goa `Consumes` DSL
	JSONDecoder = JSONFactory{}

	// JSONEncoder is the default factory used by the goa `Produces` DSL
	JSONEncoder = JSONFactory{}
)

// NewDecoder returns a new json.Decoder that satisfies goa.ResettableDecoder
func (f *JSONFactory) NewDecoder(r io.Reader) goa.ResettableDecoder {
	return codec.NewDecoder(r, &jh)
}

// NewEncoder returns a new json.Encoder that satisfies goa.ResettableDecoder
func (f *JSONFactory) NewEncoder(w io.Writer) goa.ResettableEncoder {
	return codec.NewEncoder(w, &jh)
}

// NewDecoder returns a new msgpack Decoder that satisfies goa.ResettableDecoder
func (f *MsgPackFactory) NewDecoder(r io.Reader) goa.ResettableDecoder {
	return codec.NewDecoder(r, &mh)
}

// NewEncoder returns a new msgpack Encoder that satisfies goa.ResettableDecoder
func (f *MsgPackFactory) NewEncoder(w io.Writer) goa.ResettableEncoder {
	return codec.NewEncoder(w, &mh)
}

// NewDecoder returns a new binc Decoder that satisfies goa.ResettableDecoder
func (f *BincFactory) NewDecoder(r io.Reader) goa.ResettableDecoder {
	return codec.NewDecoder(r, &bh)
}

// NewEncoder returns a new binc Encoder that satisfies goa.ResettableDecoder
func (f *BincFactory) NewEncoder(w io.Writer) goa.ResettableEncoder {
	return codec.NewEncoder(w, &bh)
}

// NewDecoder returns a new cbor Decoder that satisfies goa.ResettableDecoder
func (f *CborFactory) NewDecoder(r io.Reader) goa.ResettableDecoder {
	return codec.NewDecoder(r, &ch)
}

// NewEncoder returns a new cbor Encoder that satisfies goa.ResettableDecoder
func (f *CborFactory) NewEncoder(w io.Writer) goa.ResettableEncoder {
	return codec.NewEncoder(w, &ch)
}
