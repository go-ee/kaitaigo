package runtime

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
)

type Decoder interface {
	Read(reader io.ReadSeeker, lazy bool, ancestors ...interface{})
}

type Meta struct {
}

type TypeIO struct {
	*Stream
	Decoded    bool
	DecodeErr  error
	Meta       map[string]*Meta
	ParentBase interface{}
	RootBase   interface{}
}

func NewTypeIO(reader io.ReadSeeker, instance interface{}, ancestors ...interface{}) (ret *TypeIO) {
	ret = &TypeIO{Stream: &Stream{ReadSeeker: reader}}
	if reader == nil {
		ret.DecodeErr = errors.New("reader/decoder must not be null")
	}
	if len(ancestors) == 2 {
		ret.ParentBase = ancestors[0]
		ret.RootBase = ancestors[1]
	} else if len(ancestors) == 0 {
		ret.ParentBase = instance
		ret.RootBase = instance
	} else {
		ret.DecodeErr = errors.New("to many ancestors are given")
	}
	return
}

func (k *TypeIO) ReadBytesAsReader(n uint16) (ret io.ReadSeeker, err error) {
	var raw []byte
	if raw, err = k.ReadBytes(n); err == nil {
		ret = bytes.NewReader(raw)
	}
	return
}
