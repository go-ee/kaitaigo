// Autogenerated from KST: please remove this line if doing any edits by hand!

package type_int_unary_op

import (
	"os"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeIntUnaryOp(t *testing.T) {
	f, err := os.Open("../../../testdata/kaitai/fixed_struct.bin")
	if err != nil {
		t.Fatal(err)
	}

	var r TypeIntUnaryOp
	err = r.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 16720, r.ValueS2())
	assert.EqualValues(t, 4706543082108963651, r.ValueS8())
	tmp1 := r.UnaryS2()
	assert.EqualValues(t, -16720, tmp1)
	tmp2 := r.UnaryS8()
	assert.EqualValues(t, -4706543082108963651, tmp2)
}
