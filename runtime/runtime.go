package runtime

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
)

type KSYDecoder interface {
	Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error)
}

type TypeIO struct {
	*Stream
	ParentBase interface{}
	RootBase   interface{}
}

func NewTypeIO(reader io.ReadSeeker, instance interface{}, ancestors ...interface{}) (ret *TypeIO, err error) {
	if reader == nil {
		err = errors.New("reader/Decoder must not be null")
	}
	ret = &TypeIO{Stream: &Stream{ReadSeeker: reader}}
	if len(ancestors) == 2 {
		ret.ParentBase = ancestors[0]
		ret.RootBase = ancestors[1]
	} else if len(ancestors) == 0 {
		ret.ParentBase = instance
		ret.RootBase = instance
	} else {
		err = errors.New("to many ancestors are given")
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
