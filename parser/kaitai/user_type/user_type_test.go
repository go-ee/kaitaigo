// Autogenerated from KST: please remove this line if doing any edits by hand!

package user_type_test

import (
	"os"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserType(t *testing.T) {
	f, err := os.Open("../../../testdata/kaitai/repeat_until_s4.bin")
	if err != nil {
		t.Fatal(err)
	}

	var r UserType
	err = r.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 66, r.One().Width())
	assert.EqualValues(t, 4919, r.One().Height())
}
