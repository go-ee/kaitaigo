package runtime

import (
	"io"
)

type KSYDecoder interface {
	Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error)
}
