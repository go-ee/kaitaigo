// Autogenerated from KST: please remove this line if doing any edits by hand!

package str_literals2

import (
	"os"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrLiterals2(t *testing.T) {
	f, err := os.Open("../../../testdata/kaitai/fixed_struct.bin")
	if err != nil {
		t.Fatal(err)
	}

	var r StrLiterals2
	err = r.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	tmp1 := r.Dollar1()
	assert.EqualValues(t, "$foo", tmp1)
	tmp2 := r.Dollar2()
	assert.EqualValues(t, "${foo}", tmp2)
	tmp3 := r.Hash()
	assert.EqualValues(t, "#{foo}", tmp3)
	tmp4 := r.AtSign()
	assert.EqualValues(t, "@foo", tmp4)
}