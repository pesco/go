package ie

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func eq(s Stream, t string) bool {
	if len(t) == 0 {
		return s == Empty
	}
	return string(s.Slice().([]byte)) == t
}

func TestBind(t *testing.T) {
	u8 := Uint(BE, 1)
	it := u8.Bind(func(n_ interface{}) Iteratee {
		n := uint(n_.(uint64))
		return Uint(BE, n)
	})

	input := "\x03abcdefg"
	it, s := it.Feed(Chunk([]byte(input)))

	if !it.IsDone() {
		t.Error("should have succeeded")
		return
	}
	if !eq(s, "defg") {
		t.Error("consumed wrong; left:", s)
	}
	result := it.Result().(uint64)
	if result != 0x616263 {
		t.Errorf("wrong result; got: %#v", result)
	}
}

func TestThen(t *testing.T) {
	u32 := Uint(BE, 4)
	it := u32.Then(u32)

	input := "0123456789"
	it, s := it.Feed(Chunk([]byte(input)))

	if !it.IsDone() {
		t.Error("should have succeeded")
		return
	}
	if !eq(s, "89") {
		t.Error("consumed wrong; left:", s)
	}
	result := it.Result().(uint64)
	if result != 0x34353637 {
		t.Errorf("wrong result; got: %#v", result)
	}
}

func ExampleWrite() {
	it := Write(os.Stdout)
	it, _ = it.Feed(Chunk([]byte("hello ")))
	it, _ = it.Feed(Chunk([]byte("world\n")))
	// Output: hello world
}

func TestSeqTrivial(t *testing.T) {
	it := Seq()

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hello world!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if i.Result().([]interface{}) != nil {
			t.Error("should have returned nil, not", i.Result())
		}
		if !eq(s, "hello world!") {
			t.Error("should have consumed nothing")
		}
	}

	i, s = it.Feed(Empty)
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if i.Result().([]interface{}) != nil {
			t.Error("should have returned nil, not", i.Result())
		}
		if s != Empty {
			t.Error("should leave input unchanged")
		}
	}
}

func TestSeqSingle(t *testing.T) {
	it := Seq(String("hello world"))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hallo world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "allo world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello, world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, ", world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello wald!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "ald!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello world!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		slice := i.Result().([]interface{})
		if slice[0].(string) != "hello world" {
			t.Error("should have returned its argument")
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument")
		}
	}

	i, s = it.Feed(Chunk([]byte("hel")))
	i, s = i.Feed(Chunk([]byte("lo ")))
	i, s = i.Feed(Chunk([]byte("worl")))
	i, s = i.Feed(Chunk([]byte("d")))
	i, s = i.Feed(Chunk([]byte("!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		slice := i.Result().([]interface{})
		if slice[0].(string) != "hello world" {
			t.Error("should have returned its argument")
		}
	}
}

func TestSeqSimple(t *testing.T) {
	it := Seq(String("hello"), String(" world"))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hallo world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "allo world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello, world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, ", world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello wald!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "ald!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello world!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if fmt.Sprint(i.Result()) != "[hello  world]" {
			t.Error("should have returned its argument; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("hel")))
	i, s = i.Feed(Chunk([]byte("lo ")))
	i, s = i.Feed(Chunk([]byte("worl")))
	i, s = i.Feed(Chunk([]byte("d")))
	i, s = i.Feed(Chunk([]byte("!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if fmt.Sprint(i.Result()) != "[hello  world]" {
			t.Error("should have returned its argument; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}
}

func TestSeq(t *testing.T) {
	it := Seq(String("hello"), Byte(' '), String("world"))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hallo world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "allo world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello, world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, ", world!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello wald!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "ald!") {
		t.Error("failing Seq() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello world!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if fmt.Sprint(i.Result()) != "[hello 32 world]" {
			t.Error("should have returned its argument; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("hel")))
	i, s = i.Feed(Chunk([]byte("lo ")))
	i, s = i.Feed(Chunk([]byte("worl")))
	i, s = i.Feed(Chunk([]byte("d")))
	i, s = i.Feed(Chunk([]byte("!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if fmt.Sprint(i.Result()) != "[hello 32 world]" {
			t.Error("should have returned its argument; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}
}

func TestSeq_(t *testing.T) {
	it := Seq_(String("hello"), Byte(' '), String("world"))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hallo world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "allo world!") {
		t.Error("failing Seq_() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello, world!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, ", world!") {
		t.Error("failing Seq_() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello wald!")))
	if i.Err() == nil {
		t.Error("should have failed")
	} else if !eq(s, "ald!") {
		t.Error("failing Seq_() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello world!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if i.Result() != nil {
			t.Error("should have returned nil; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("hel")))
	i, s = i.Feed(Chunk([]byte("lo ")))
	i, s = i.Feed(Chunk([]byte("worl")))
	i, s = i.Feed(Chunk([]byte("d")))
	i, s = i.Feed(Chunk([]byte("!")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if i.Result() != nil {
			t.Error("should have returned nil; got:", i.Result())
		}
		if !eq(s, "!") {
			t.Error("should have consumed its argument; left:", s)
		}
	}
}

func TestMany(t *testing.T) {
	it := Many([]byte(nil), Byte('a'))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("aaaa")))
	if i.IsDone() {
		t.Error("Many() terminated prematurely; result:", i.Result())
	} else {
		if s != Empty {
			t.Error("consumed wrong; left:", s)
		}
		i, s = i.Feed(Chunk([]byte("aa")))
		if s != Empty {
			t.Error("consumed wrong; left:", s)
		}
		i, s = i.Feed(End)
		if !i.IsDone() {
			t.Error("should have succeeded")
		} else {
			r := string(i.Result().([]byte))
			if r != "aaaaaa" {
				t.Error("should have returned \"aaaaaa\"; got:", r)
			}
			if s != End {
				t.Error("consumed wrong; left:", s)
			}
		}
	}

	i, s = it.Feed(Chunk([]byte("aaaabb")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		r := string(i.Result().([]byte))
		if r != "aaaa" {
			t.Error("should have returned \"aaaa\"; got:", r)
		}
		if !eq(s, "bb") {
			t.Error("consumed wrong; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("bb")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		r := string(i.Result().([]byte))
		if r != "" {
			t.Error("should have returned \"\"; got:", r)
		}
		if !eq(s, "bb") {
			t.Error("consumed wrong; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("")))
	i, s = it.Feed(End)
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		r := string(i.Result().([]byte))
		if r != "" {
			t.Error("should have returned \"\"; got:", r)
		}
		if s != End {
			t.Error("consumed wrong; left:", s)
		}
	}
}

func TestMany_(t *testing.T) {
	it := Many_(Byte('a'))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("aaaa")))
	if i.IsDone() {
		t.Error("Many() terminated prematurely; result:", i.Result())
	} else {
		if s != Empty {
			t.Error("consumed wrong; left:", s)
		}
		i, s = i.Feed(Chunk([]byte("aa")))
		if s != Empty {
			t.Error("consumed wrong; left:", s)
		}
		i, s = i.Feed(End)
		if !i.IsDone() {
			t.Error("should have succeeded")
		} else {
			if i.Result() != nil {
				t.Error("should have returned nil; got:", i.Result())
			}
			if s != End {
				t.Error("consumed wrong; left:", s)
			}
		}
	}

	i, s = it.Feed(Chunk([]byte("aaaabb")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if i.Result() != nil {
			t.Error("should have returned nil; got:", i.Result())
		}
		if !eq(s, "bb") {
			t.Error("consumed wrong; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("bb")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		r := i.Result()
		if r != nil {
			t.Error("should have returned nil; got:", r)
		}
		if !eq(s, "bb") {
			t.Error("consumed wrong; left:", s)
		}
	}

	i, s = it.Feed(Chunk([]byte("")))
	i, s = it.Feed(End)
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		r := i.Result()
		if r != nil {
			t.Error("should have returned nil; got:", r)
		}
		if s != End {
			t.Error("consumed wrong; left:", s)
		}
	}
}

func TestOptional(t *testing.T) {
	it := Optional(String("XYZ"))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("abc")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if !eq(s, "abc") {
			t.Error("consumed wrong; left:", s)
		}
		if i.Result() != nil {
			t.Error("should have returned nil; got:", i.Result())
		}
	}

	i, s = it.Feed(Chunk([]byte("Xbc")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if !eq(s, "Xbc") {
			t.Error("consumed wrong; left:", s)
		}
		if i.Result() != nil {
			t.Error("should have returned nil; got:", i.Result())
		}
	}

	// must send this iteratee enough input to decide (no rewind)
	i, s = it.Feed(Chunk([]byte("XY")))
	if !i.IsCont() {
		t.Error("should have suspended")
	}
	if s != Empty {
		t.Error("consumed wrong; left:", s)
	}
	func() {
		defer func() {
			if msg := recover(); msg != nil {
				if !strings.Contains(msg.(string), "lookahead") {
					t.Error("expected lookahead error; got:", msg)
				}
			} else {
				t.Error("should have panicked")
			}
		}()
		i, s = i.Feed(End)
	}()

	i, s = it.Feed(Chunk([]byte("XYZabc")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if !eq(s, "abc") {
			t.Error("consumed wrong; left:", s)
		}
		if i.Result().(string) != "XYZ" {
			t.Error("should have returned \"XYZ\"; got:", i.Result())
		}
	}
}

func TestSkip(t *testing.T) {
	testcase := func(it Iteratee, input string, rest string) {
		it, s := it.Feed(Chunk(input))
		if !it.IsDone() {
			t.Error("should have succeeded")
			return
		}
		if it.Result() != nil {
			t.Error("wrong result (should have been nil); got:", it.Result())
		}
		if !eq(s, rest) {
			t.Error("consumed wrong; left:", s)
		}
	}

	testcase_multi := func(it Iteratee, in1, in2 string, rest string) {
		it, s := it.Feed(Chunk(in1))
		if !it.IsCont() {
			t.Error("should have suspended")
			return
		}
		it, s = it.Feed(Chunk(in2))
		if !it.IsDone() {
			t.Error("should have succeeded")
			return
		}
		if it.Result() != nil {
			t.Error("wrong result (should have been nil); got:", it.Result())
		}
		if !eq(s, rest) {
			t.Error("consumed wrong; left:", s)
		}
	}

	testcase_fail := func(it Iteratee, input string) {
		it, _ = it.Feed(Chunk(input))
		if !it.IsCont() {
			t.Error("should have suspended")
			return
		}
		it, _ = it.Feed(End)
		if it.IsDone() {
			t.Error("should not have succeeded; got", it.Result())
			return
		}
	}

	testcase(Skip(5), "0123456789", "56789")
	testcase(Skip(3), "0123456789", "3456789")
	testcase(Skip(0), "0123456789", "0123456789")
	testcase(Skip(-23), "0123456789", "0123456789")
	testcase(Skip(10), "0123456789", "")

	testcase_multi(Skip(5), "", "0123456789", "56789")
	testcase_multi(Skip(5), "012", "3456789", "56789")
	testcase_multi(Skip(5), "0123", "456789", "56789")

	testcase_fail(Skip(5), "")
	testcase_fail(Skip(5), "012")
	testcase_fail(Skip(5), "0123")
}
