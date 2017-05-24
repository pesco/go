package ie

import (
	"testing"
	"os"
	"strings"
	
	"github.com/pesco/go/monad"
)

func ExamplePass() {
	enum := Read(strings.NewReader("hallo ")).Append(
		    Read(strings.NewReader("welt!\n")))
	inner := Write(os.Stdout)
	outer := Pass(inner)
	io := enum(outer).(monad.IO)
	io()
	// Output: hallo welt!
}

func ExamplePipe() {
	enum := Read(strings.NewReader("hallo ")).Append(
		    Read(strings.NewReader("welt!\n")))
	enum = enum.Pipe(Pass)
	inner := Write(os.Stdout)
	io := enum(inner).(monad.IO)
	io()
	// Output: hallo welt!
}

func TestEnumerateeAppend(t *testing.T) {
	eol    := Choice(Byte('\n'), EndOfInput)
	white  := OneOf([]byte(" \t"))
	prefix := Byte('>').Then(Choice(eol, Many1_(white)))

	line    := BreakAfter([]byte("\n"))
	qline   := Prefix(prefix, line).ToEnumerator()
	quoted  := qline.Append(qline)

	it := quoted(String("abc\ndef\n")).(Iteratee).Fuse()
	result := parse(it, "> abc\n> def\n")
	if result.(string) != "abc\ndef\n" {
		t.Error("wrong result; got:", result)
	}
}

func TestRepeat(t *testing.T) {
	eol    := Choice(Byte('\n'), EndOfInput)
	white  := OneOf([]byte(" \t"))
	prefix := Byte('>').Then(Choice(eol, Many1_(white)))

	line    := BreakAfter([]byte("\n"))
	qline   := Prefix(prefix, line)
	quoted  := Repeat(qline)

	var it Iteratee
	var result interface{}

	it = quoted(String("abc\ndef\nghi\n")).Fuse()
	result = parse(it, "> abc\n>   def\n> ghi\n>> xyz")
	if result.(string) != "abc\ndef\nghi\n" {
		t.Error("wrong result; got:", result)
	}

	it = quoted(Many([]byte(nil),Any)).Fuse()
	result = parse(it, "wurst")
	if len(result.([]byte)) != 0 {
		t.Error("expected empty result; got:", result)
	}
}

func TestRepeat1(t *testing.T) {
	eol    := Choice(Byte('\n'), EndOfInput)
	white  := OneOf([]byte(" \t"))
	prefix := Byte('>').Then(Choice(eol, Many1_(white)))

	line    := BreakAfter([]byte("\n"))
	qline   := Prefix(prefix, line)
	quoted  := Repeat1(qline)

	it := quoted(String("abc\ndef\nghi\n")).(Iteratee).Fuse()
	result := parse(it, "> abc\n>   def\n> ghi\n>> xyz")
	if result.(string) != "abc\ndef\nghi\n" {
		t.Error("wrong result; got:", result)
	}
}
