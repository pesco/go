package ie

import (
	"testing"
	"os"
	"strings"
	"bytes"

	"github.com/pesco/go/monad"
)


func ExampleRead() {
	enum := Read(strings.NewReader("hallo welt!\n"))
	it := Write(os.Stdout)
	io := enum(it).(monad.IO)
	io()
	// Output: hallo welt!
}

func ExampleAppend() {
	a := Read(strings.NewReader("hallo "))
	b := Read(strings.NewReader("welt"))
	c := Read(strings.NewReader("!\n"))
	var abc Enumerator = a.Append(b).Append(c)

	io := abc(Write(os.Stdout)).(monad.IO)
	io()
	// Output: hallo welt!
}

func TestSeek(t *testing.T) {
	u32 := Uint(BE, 4)

	testcase := func(it Iteratee, result uint64) {
		enum := SeekableRead(bytes.NewReader([]byte("0123456789")))
		it = enum(it).(monad.IO)().(Iteratee)

		if !it.IsDone() {
			t.Error("should have succeeded; err:", it.Err())
			return
		}
		r := it.Result().(uint64)
		if r != result {
			t.Errorf("wrong result; expected %#v, got %#v", result, r)
		}
	}

	testcase(u32.Then(Stop(Seek{3}, u32.k)), 0x33343536)
	testcase(Raise(Seek{2}).Then(u32), 0x32333435)
	testcase(u32.Then(Raise(Seek{2})).Then(u32), 0x32333435)
}
