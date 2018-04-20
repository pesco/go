package ie

import (
	"reflect"
	"testing"

	"github.com/pesco/go/monad"
)

func parse(it Iteratee, s string) interface{} {
	return EnumString(s)(it).(monad.IO)().(Iteratee).Run()
}

func biteq(s Stream, bs []byte, bitorder Endianness, offs uint8) bool {
	return string(s.Bytes()) == string(bs) &&
		s.Endian() == bitorder &&
		s.Offset() == offs
}

func TestByte(t *testing.T) {
	it := Byte('x')

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hello")))
	if i.Err() == nil {
		t.Error("should have failed")
	}
	if !eq(s, "hello") {
		t.Error("failing Byte() should have consumed nothing")
	}

	i, s = it.Feed(Chunk([]byte("xyz")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	}
	if !eq(s, "yz") {
		t.Error("Byte('x') should have consumed 'x'")
	}

	i, s = it.Feed(Chunk([]byte{}))
	if i.IsDone() {
		t.Error("stopped on empty chunk")
	}
	if i.Err() != nil {
		t.Error("error on empty chunk")
	}
	i, s = i.Feed(Chunk([]byte("x")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	}
	if s != Empty {
		t.Error("Byte('x') should have consumed 'x'")
	}
}

func TestString(t *testing.T) {
	it := String("hello")

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("hallo")))
	if i.Err() == nil {
		t.Error("should have failed")
	}
	if !eq(s, "allo") {
		t.Error("failing String() consumed wrong; left:", s)
	}

	i, s = it.Feed(Chunk([]byte("hello world")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	}
	if !eq(s, " world") {
		t.Error("String(\"hello\") should have consumed \"hello\"")
	}

	i, s = it.Feed(Chunk([]byte{}))
	if i.IsDone() {
		t.Error("stopped on empty chunk")
	}
	if i.Err() != nil {
		t.Error("error on empty chunk")
	}
	i, s = i.Feed(Chunk([]byte("hello")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	}
	if s != Empty {
		t.Error("String(\"hello\") should have consumed \"hello\"")
	}

	i, s = it.Feed(Chunk([]byte("hel")))
	i, s = i.Feed(Chunk([]byte("lo ")))
	i, s = i.Feed(Chunk([]byte("world")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	}
}

func TestOneOf(t *testing.T) {
	it := Many([]byte(nil), OneOf([]byte("abc")))

	var i Iteratee
	var s Stream

	i, s = it.Feed(Chunk([]byte("aaaaaabcbcbccccbb-")))
	if !i.IsDone() {
		t.Error("should have succeeded")
	} else {
		if !eq(s, "-") {
			t.Error("consumed wrong; left:", s)
		}
		if string(i.Result().([]byte)) != "aaaaaabcbcbccccbb" {
			t.Error("wrong result; got:", i.Result())
		}
	}
}

func testBits(bitorder Endianness, t *testing.T) {
	var it, i Iteratee
	var s Stream
	var result uint64

	testcase := func(bitorder Endianness, n uint8,
		input string, offs uint8,
		expect uint64,
		rest string, roffs uint8) {
		it = Bits(bitorder, n)
		i, s = it.Feed(BitChunk([]byte(input), bitorder, offs))
		if !i.IsDone() {
			t.Error("should have succeeded")
			return
		}
		result = i.Result().(uint64)
		if result != expect {
			t.Errorf("wrong result; got: 0x%x", result)
		}
		if !biteq(s, []byte(rest), bitorder, roffs) {
			t.Error("consumed wrong; left:", s)
		}
	}

	// multi-chunk input
	testcase_multi := func(bitorder Endianness, n uint8,
		input1 string, offs uint8, input2 string,
		expect uint64,
		rest string, roffs uint8) {
		it = Bits(bitorder, n)
		i, s = it.Feed(BitChunk([]byte(input1), bitorder, offs))
		if !i.IsCont() {
			t.Error("should have suspended")
			return
		}
		if s != Empty {
			t.Error("consumed wrong; left:", s)
		}
		i, s = i.Feed(BitChunk([]byte(input2), bitorder, 0))
		if !i.IsDone() {
			t.Error("should have succeeded")
			return
		}
		result = i.Result().(uint64)
		if result != expect {
			t.Errorf("wrong result; got: 0x%x", result)
		}
		if !biteq(s, []byte(rest), bitorder, roffs) {
			t.Error("consumed wrong; left:", s)
		}
	}

	// premature end of input
	testcase_end := func(bitorder Endianness, n uint8, input string, offs uint8) {
		it = Bits(BE, 23)
		i, s = it.Feed(BitChunk([]byte{0x12, 0x34}, BE, 4))
		if !i.IsCont() {
			t.Error("should have suspended")
			return
		}
		i, s = i.Feed(End)
		if !i.IsStop() {
			t.Error("should have failed")
		}
	}

	testcase(BE, 4, "\x12\x34\x56\x78\x9a", 3, 0x9, "\x12\x34\x56\x78\x9a", 7)
	testcase(BE, 12, "\x12\x34\x56\x78\x9a", 0, 0x123, "\x34\x56\x78\x9a", 4)
	testcase(BE, 23, "\x12\x34\x56\x78\x9a", 4, 0x11a2b3, "\x78\x9a", 3)
	testcase_multi(BE, 23, "\x12\x34", 4, "\x56\x78\x9a", 0x11a2b3, "\x78\x9a", 3)
	testcase_end(BE, 23, "\x12\x34", 4)

	testcase(LE, 4, "\x12\x34\x56\x78\x9a", 3, 0x2, "\x12\x34\x56\x78\x9a", 7)
	testcase(LE, 12, "\x12\x34\x56\x78\x9a", 0, 0x412, "\x34\x56\x78\x9a", 4)
	testcase(LE, 23, "\x12\x34\x56\x78\x9a", 4, 0x056341, "\x78\x9a", 3)
	testcase_multi(LE, 23, "\x12\x34", 4, "\x56\x78\x9a", 0x056341, "\x78\x9a", 3)
	testcase_end(LE, 23, "\x12\x34", 4)
}

func TestUint(t *testing.T) {
	var it, i Iteratee
	var s Stream
	var result uint64

	testcase := func(byteorder Endianness, n uint,
		input string, expect uint64, rest string) {
		it = Uint(byteorder, n)
		i, s = it.Feed(Chunk(input))
		if !i.IsDone() {
			t.Error("should have succeeded")
			return
		}
		result = i.Result().(uint64)
		if result != expect {
			t.Errorf("wrong result; got: 0x%x", result)
		}
		if !eq(s, rest) {
			t.Error("consumed wrong; left:", s)
		}
	}

	testcase_multi := func(byteorder Endianness, n uint,
		in1, in2 string, expect uint64, rest string) {
		it = Uint(byteorder, n)
		i, s = it.Feed(Chunk(in1))
		if !i.IsCont() {
			t.Error("should have suspended")
			return
		}
		i, s = i.Feed(Chunk(in2))
		if !i.IsDone() {
			t.Error("should have succeeded")
			return
		}
		result = i.Result().(uint64)
		if result != expect {
			t.Errorf("wrong result; got: 0x%x", result)
		}
		if !eq(s, rest) {
			t.Error("consumed wrong; left:", s)
		}
	}

	testcase_end := func(byteorder Endianness, n uint, input string) {
		it = Uint(byteorder, n)
		i, s = it.Feed(Chunk(input))
		if !i.IsCont() {
			t.Error("should have suspended")
			return
		}
		i, s = i.Feed(End)
		if !i.IsStop() {
			t.Error("should have failed")
		}
	}

	testcase(LE, 1, "\x12\x34\x56", 0x12, "\x34\x56")
	testcase(LE, 2, "\x12\x34\x56", 0x3412, "\x56")
	testcase(LE, 3, "\x12\x34\x56", 0x563412, "")
	testcase(LE, 4, "\x12\x34\x56\x78\x9a", 0x78563412, "\x9a")
	testcase(LE, 6, "\x12\x34\x56\x78\x9a\xbc", 0xbc9a78563412, "")
	testcase(LE, 8, "\x12\x34\x56\x78\x9a\xbc\xde\xf0", 0xf0debc9a78563412, "")
	testcase_multi(LE, 4, "\x12\x34", "\x56\x78\x9a", 0x78563412, "\x9a")
	testcase_end(LE, 4, "\x12\x34")

	testcase(BE, 1, "\x12\x34\x56", 0x12, "\x34\x56")
	testcase(BE, 2, "\x12\x34\x56", 0x1234, "\x56")
	testcase(BE, 3, "\x12\x34\x56", 0x123456, "")
	testcase(BE, 4, "\x12\x34\x56\x78\x9a", 0x12345678, "\x9a")
	testcase(BE, 6, "\x12\x34\x56\x78\x9a\xbc", 0x123456789abc, "")
	testcase(BE, 8, "\x12\x34\x56\x78\x9a\xbc\xde\xf0", 0x123456789abcdef0, "")
	testcase_multi(BE, 4, "\x12\x34", "\x56\x78\x9a", 0x12345678, "\x9a")
	testcase_end(BE, 4, "\x12\x34")
}

type TS1 struct {
	A uint16
	B [3]uint8
	C uint32
}
type TS2 struct {
	A uint16
	_ [3]uint8
	C uint32
}

func TestStruct(t *testing.T) {
	testcase := func(ptr interface{}, input string, result interface{}, rest string) {
		it := Struct(LE, ptr)

		it, s := it.Feed(Chunk(input))
		if !it.IsDone() {
			t.Error("should have succeeded; err:", it.Err())
			return
		}
		if !eq(s, rest) {
			t.Errorf("consumed wrong; expected %q, got %q", rest, s)
		}
		r := it.Result()
		if reflect.TypeOf(r) != reflect.TypeOf(result) {
			t.Errorf("wrong result type; expected %T, got %T", result, r)
			return
		}
		if !reflect.DeepEqual(r, result) {
			t.Errorf("wrong result; expected %#v, got %#v", result, r)
		}
	}

	testcase((*uint8)(nil), "01234", uint8(0x30), "1234")
	testcase((*uint16)(nil), "01234", uint16(0x3130), "234")
	testcase((*uint32)(nil), "01234", uint32(0x33323130), "4")
	testcase((*uint64)(nil), "0123456789", uint64(0x3736353433323130), "89")
	testcase((*[3]uint16)(nil), "0123456789", [3]uint16{0x3130, 0x3332, 0x3534}, "6789")
	testcase((*TS1)(nil), "0123456789", TS1{0x3130, [3]uint8{0x32, 0x33, 0x34}, 0x38373635}, "9")
	testcase((*TS2)(nil), "0123456789", TS2{0x3130, [3]uint8{}, 0x38373635}, "9")
}
