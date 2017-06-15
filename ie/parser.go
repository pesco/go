package ie

import (
	"fmt"
	"bytes"
	"reflect"
)


type NoMatch struct {Expect string}
func (e NoMatch) Error() string {return (e.Expect + ": no match")}


// primitive parsers...

var EndOfInput Iteratee = Cont(k_eof)
func k_eof(s Stream) (Iteratee, Stream) {
	if s == Empty {
		return Cont(k_eof), s
	}
	if s != End {
		return Fail(NoMatch{"expected end of input"}), s
	}
	return Done(nil), s
}

// an alias for Head that fits better when building parsers
var Any Iteratee = Head

func Byte(b byte) (this Iteratee) {
	this = Cont(func(s Stream) (Iteratee, Stream) {
		if s == End {
			return Fail(NoMatch{fmt.Sprintf("%q (unexpected end of input)", b)}), s
		}
		if s == Empty {
			return this, s
		}
		slice := s.Slice().([]byte)
		if slice[0] != b {
			return Fail(NoMatch{fmt.Sprintf("%q (unexpected %q)", b, slice[0])}), s
		}
		return Done(b), Chunk(slice[1:])
	})
	return
}

func String(x string) Iteratee {
	return string_(x).Then(Done(x))
}

// nil result
func string_(x string) (this Iteratee) {
	if x == "" {
		return Done(nil)
	}
	this = Cont(func(s Stream) (Iteratee, Stream) {
		if s == End {
			return Fail(NoMatch{fmt.Sprintf("%q (unexpected end of input)", x)}), s
		}
		if s == Empty {
			return this, s
		}
		slice := s.Slice().([]byte)
		for i := 0; i < len(x); i++ {
			if i == len(slice) {
				return string_(x[i:]), Empty
			}
			if slice[i] != x[i] {
				return Fail(NoMatch{fmt.Sprintf("%q (unexpected %q)", x, slice[i])}),
				       Chunk(slice[i:])
			}
		}
		return Done(nil), Chunk(slice[len(x):])
	})
	return
}

func OneOf(set []byte) (this Iteratee) {
	this = Cont(func(s Stream) (Iteratee, Stream) {
		if s == End {
			return Fail(NoMatch{"unexpected end of input"}), s
		}
		if s == Empty {
			return this, s
		}
		bs := s.Slice().([]byte)
		if bytes.IndexByte(set, bs[0]) == -1 {
			return Fail(NoMatch{fmt.Sprintf("unexpected %q", bs[0])}), s
		}
		return Done(bs[0]), Chunk(bs[1:])
	})
	return
}

// XXX todo NoneOf

func Range(it Iteratee, min,max uint64) Iteratee {
	return Validate(it, func(x_ interface{}) bool {
		x := x_.(uint64)
		return x >= min && x <= max
	})
}

// n-byte numbers, general case
func Uint(byteorder Endianness, n uint) Iteratee {
	if n > 8 {
		panic("Uint() called with n>8")
	}
	if n == 0 {
		return Done(uint64(0))
	}

	conv := func(r uint64) interface{} {return r}	// id
	return ui(byteorder, n, conv)
}
func ui(byteorder Endianness, n uint,
        conv func(r uint64) interface{}) Iteratee {
	var addbyte func(r uint64, b byte, pos uint) uint64

	if byteorder == LE {
		addbyte = func(r uint64, b byte, pos uint) uint64 {
			return r | (uint64(b) << (8*pos))
		}
	} else {
		addbyte = func(r uint64, b byte, _ uint) uint64 {
			return (r << 8) | uint64(b)
		}
	}

	var iter func(res uint64, pos uint) Iteratee
	iter = func(res uint64, pos uint) (this Iteratee) {
		return Cont(func(s Stream) (Iteratee, Stream) {
			if s == End {
				return Fail(NoMatch{fmt.Sprintf("Uint(%d): unexpected end of input", n)}), s
			}
			if s == Empty {
				return this, s
			}

			bs := s.Slice().([]byte)
			m := n-pos
			if uint(len(bs)) < m {
				m = uint(len(bs))
			}

			r := res
			p := pos
			for i := uint(0); i < m; i++ {
				r = addbyte(r, bs[i], p)
				p++
			}
			bs = bs[m:]

			if p < n {
				return iter(r, p), Empty
			}
			return Done(conv(r)), Chunk(bs)
		})
	}

	return iter(0, 0)
}

// n-byte numbers, type-specialized:

var Uint8 Iteratee = Head

func Uint16(byteorder Endianness) Iteratee {
	conv := func(r uint64) interface{} {return uint16(r)}
	return ui(byteorder, 2, conv)
}

func Uint32(byteorder Endianness) Iteratee {
	conv := func(r uint64) interface{} {return uint32(r)}
	return ui(byteorder, 4, conv)
}

func Uint64(byteorder Endianness) Iteratee {
	conv := func(r uint64) interface{} {return r}	// id
	return ui(byteorder, 8, conv)
}

// n-bit numbers, general case; consumes a bit stream!
func Bits(bitorder Endianness, n uint8) Iteratee {
	// this is implemented analogously to Uint() but complicated by the
	// possibility of partial bytes in the input.

	if n > 64 {
		panic("Bits() called with n>64")
	}
	if n == 0 {
		return Done(uint64(0))
	}

	var (
		addbits func(r uint64, pos uint8, b byte, n,offset uint8) uint64
		addfirst func(r uint64, pos uint8, b byte, n,offset uint8) uint64
		addbyte func(r uint64, pos uint8, b byte) uint64
		addlast func(r uint64, pos uint8, b byte, n uint8) uint64
	)

	if bitorder == LE {
		addbits = func(r uint64, pos uint8, b byte, n,offset uint8) uint64 {
			return r | (uint64((b >> offset) & ^(0xFF << n)) << pos)
		}
		addfirst = func(r uint64, pos uint8, b byte, n,offset uint8) uint64 {
			return r | (uint64(b >> offset) << pos)
		}
		addbyte = func(r uint64, pos uint8, b byte) uint64 {
			return r | (uint64(b) << pos)
		}
		addlast = func(r uint64, pos uint8, b byte, n uint8) uint64 {
			return r | (uint64(b & ^(0xFF << n)) << pos)
		}
	} else {
		addbits = func(r uint64, _ uint8, b byte, n,offset uint8) uint64 {
			return (r << n) | (uint64(b << offset) << n >> 8)
		}
		addfirst = addbits
		addbyte = func(r uint64, _ uint8, b byte) uint64 {
			return (r << 8) | uint64(b)
		}
		addlast = func(r uint64, _ uint8, b byte, n uint8) uint64 {
			return (r << n) | (uint64(b) << n >> 8)
		}
	}

	var iter func(res uint64, pos uint8, n uint8) Iteratee
	iter = func(res uint64, pos uint8, n uint8) (this Iteratee) {
		this = Cont(func(s Stream) (Iteratee, Stream) {
			if s == End {
				return Fail(NoMatch{fmt.Sprintf("Bits(%d): unexpected end of input", n)}), s
			}
			if s == Empty {
				return this, s
			}

			bs := s.Bytes()
			avail := 8 - s.offset
			r := res
			p := pos

			if s.bitorder != bitorder {
				panic("fed wrong-endian chunk to Bits()")
				// XXX read other-endian and reverse?
			}

			// use the (partial) first byte
			if n < avail {
				r = addbits(r, p, bs[0], n, s.offset)
					// (LE) r |= uint64((bs[0] >> s.offset) & ^(0xFF << n)) << p
					// (BE) r = (r << n) | (uint64(bs[0] << s.offset) << n >> 8)
				return Done(r), BitChunk(bs, s.bitorder, s.offset+n)
			}
			r = addfirst(r, p, bs[0], avail, s.offset)
				// r = addbits(r, p, bs[0], avail, s.offset)
				// (LE) r |= uint64(bs[0] >> s.offset) << p
				// (BE) r = (r << avail) | (uint64(bs[0] << s.offset) << avail >> 8)
			n -= avail
			p += avail
			bs = bs[1:]

			// consume full bytes as needed/available
			m := int(n)/8	// number of bytes to consume
			if len(bs) < m {
				m = len(bs)
			}
			for i := 0; i < m; i++ {
				r = addbyte(r, p, bs[i])
					// r = addbits(r, p, bs[i], 8, 0)
					// (LE) r = r | (uint64(bs[i]) << p)
					// (BE) r = (r << 8) | uint64(bs[i])
				p += 8
			}
			n -= uint8(m)*8
			bs = bs[m:]

			if n == 0 {
				return Done(r), BitChunk(bs, s.bitorder, 0)
			}

			// suspend if out of input
			if len(bs) == 0 {
				return iter(r, p, n), Empty
			}

			// consume partial last byte
			r = addlast(r, p, bs[0], n)
				// r = addbits(r, p, bs[0], n, 0)
				// (LE) r |= uint64(bs[0] & ^(0xFF << n)) << p
				// (BE) r = (r << n) | (uint64(bs[0]) << n >> 8)
			return Done(r), BitChunk(bs, s.bitorder, n)
		})
		return
	}

	return iter(0, 0, n)
}

// Iteratee equivalent of binary.Read.
// ptr must be a pointer to the data structure to be filled;
// if passed a typed (!) nil, the result will be allocated.
// Supported types are combinations of fixed-size numeric types, arrays, and
// structs.
func Struct(byteorder Endianness, ptr interface{}) Iteratee {
	vptr := reflect.ValueOf(ptr)
	Tptr := vptr.Type()
	if Tptr.Kind() != reflect.Ptr {
		panic("Struct: expects pointer argument")
	}
	if vptr.IsNil() {
		vptr = reflect.New(Tptr.Elem())
	}
	return StructV(byteorder, vptr.Elem());
}
func StructV(bo Endianness, v reflect.Value) Iteratee {
	T := v.Type()

	setv := func(x interface{}) Iteratee {
		v.Set(reflect.ValueOf(x))
		return Done(v.Interface())
	}

	switch(T.Kind()) {
	case reflect.Uint8:  return Uint8.Bind(setv)
	case reflect.Uint16: return Uint16(bo).Bind(setv)
	case reflect.Uint32: return Uint32(bo).Bind(setv)
	case reflect.Uint64: return Uint64(bo).Bind(setv)
	case reflect.Array:  return fillarray(bo, v, 0)
	case reflect.Struct: return fillstruct(bo, v, 0)
	default:
		panic(fmt.Sprintf("Struct: type %v not supported", T))
	}
}
func fillarray(bo Endianness, v reflect.Value, i int) Iteratee {
	if i >= v.Len() {
		return Done(v.Interface())
	}
	elem := StructV(bo, v.Index(i))
	k := func(_ interface{}) Iteratee {return fillarray(bo, v, i+1)}
	return elem.Bind(k)
}
func fillstruct(bo Endianness, v reflect.Value, i int) Iteratee {
	if i >= v.NumField() {
		return Done(v.Interface())
	}
	var elem Iteratee
	field := v.Type().Field(i)

	if field.Name == "_" {
		// skip blank (_) fields like binary.Read
		elem = Skip(StructSize(field.Type))
	} else {
		elem = StructV(bo, v.Field(i))
	}

	k := func(_ interface{}) Iteratee {return fillstruct(bo, v, i+1)}
	return elem.Bind(k)
}
func StructSize(T reflect.Type) int {
	switch(T.Kind()) {
	case reflect.Uint8:  return 1
	case reflect.Uint16: return 2
	case reflect.Uint32: return 4
	case reflect.Uint64: return 8
	case reflect.Array:  return T.Len() * StructSize(T.Elem())
	case reflect.Struct:
		size := 0
		for i := 0; i < T.NumField(); i++ {
			size += StructSize(T.Field(i).Type)
		}
		return size
	default:
		panic(fmt.Sprintf("StructSize: type %v not supported", T))
	}
}


// combinators...

// returns results as a []interface{}
func Seq(its ...Iteratee) Iteratee {
	if len(its) == 0 {
		return Done([]interface{}(nil))
	}
	return seq1(0, its)
}

func seq1(i int, its []Iteratee) Iteratee {
	if i >= len(its) {
		slice := make([]interface{}, len(its))
		return Done(slice)
	}

	return its[i].Bind(func(x interface{}) Iteratee {
		return seq1(i+1, its).Bind(func(sl interface{}) Iteratee {
			slice := sl.([]interface{})
			slice[i] = x
			return Done(slice)
		})
	})
}

// discards results
func Seq_(its ...Iteratee) Iteratee {
	if len(its) == 0 {
		return Done(nil)
	}

	return its[0].Bind(func(_ interface{}) Iteratee {
		return Seq_(its[1:]...)
	})
}

// run all arguments in parallel, return first result found
func Choice(its ...Iteratee) Iteratee {
	if len(its) == 0 {
		return Fail(NoMatch{"Choice"})
	}

	return Cont(func(s Stream) (Iteratee, Stream) {
		rest := []Iteratee(nil)
		for _, it := range its {
			it, t := it.Feed(s)
			if it.k == nil {
				return Done(it.result), t
			}
			if it.err == nil {
				rest = append(rest, it)
			}
		}
		return Choice(rest...), Empty
	})
}

// run all arguments in parallel, return the *leftmost* match
func OChoice(its ...Iteratee) Iteratee {
	return ochoice(its, false)
}
func ochoice(its []Iteratee, commit bool) Iteratee {
	if len(its) == 0 {
		if commit {
			// we had a success earlier (that we cannot go back to now) but
			// couldn't tell at the time that all other choices would fail.
			// (cf. comment below)
			// NB: this case is a panic because whether it is triggered depends
			// on the placement of chunk boundaries. therefore, unless the
			// programmer ensures that an OChoice always has enough lookahead,
			// the language accepted by the iteratee is not well-defined.
			panic("OChoice: lookahead needed")
		}
		return Fail(NoMatch{"OChoice"})
	}

	return Cont(func(s Stream) (Iteratee, Stream) {
		rest := []Iteratee(nil)
		for _, it := range its {
			it, t := it.Feed(s)
			if it.k == nil {	// is done
				if rest == nil {
					// all previous iteratees failed -> match
					return Done(it.result), t
				} else if (t != Empty && t != End) {
					// this iteratee succeeded and left some input, but others
					// before it are suspended waiting for another chunk.
					// since we don't support rewinding the input stream, we
					// must at this point commit to match one of the suspended
					// iteratees.
					// so we break out of the loop here and signal this condition
					// to our future self (see error case above).
					return ochoice(rest, true), Empty
				}
			}
			if it.err == nil {
				rest = append(rest, it)
			}
		}
		if s != End {
			s = Empty
		}
		return ochoice(rest, commit), s
	})
}

// appends results to 'slice' which must have type []T
// note: it is allowed to pass an appropriately-typed nil, e.g.:
//
//   Many([]error(nil), getError)
//
// think of it as a form of poor man's parametric types
func Many(slice interface{}, it Iteratee) Iteratee {
	return OChoice(Many1(slice, it), Done(slice))
}

func Many1(slice interface{}, it Iteratee) Iteratee {
	return it.Bind(func(x interface{}) Iteratee {
		vslice := reflect.ValueOf(slice)
		vslice = reflect.Append(vslice, reflect.ValueOf(x))
		return Many(vslice.Interface(), it)
	})
}

// like Many but discards results
func Many_(it Iteratee) Iteratee {
	return Optional(Many1_(it))
}

// like Many1 but discards results
func Many1_(it Iteratee) Iteratee {
	return it.Bind(func(interface{}) Iteratee {return Many_(it)})
}

// a variant of Many that requires all input to match the given
// iteratee. passes the error that 'it' stopped with.
func ManyEnd(slice interface{}, it Iteratee) Iteratee {
	return Cont(func(s Stream) (Iteratee, Stream) {
		if s == End {
			return Done(slice), s
		}
		return Many1End(slice, it).Feed(s)
	})
}

func Many1End(slice interface{}, it Iteratee) Iteratee {
	return it.Bind(func(x interface{}) Iteratee {
		vslice := reflect.ValueOf(slice)
		vslice = reflect.Append(vslice, reflect.ValueOf(x))
		return ManyEnd(vslice.Interface(), it)
	})
}

func Optional(it Iteratee) Iteratee {
	return OChoice(it, Done(nil))
}

func Times(n int, slice interface{}, it Iteratee) Iteratee {
	if n <= 0 {
		return Done(slice)
	}
	return it.Bind(func(x interface{}) Iteratee {
		vslice := reflect.ValueOf(slice)
		vslice = reflect.Append(vslice, reflect.ValueOf(x))
		return Times(n-1, vslice.Interface(), it)
	})
}

func Times_(n int, it Iteratee) Iteratee {
	if n <= 0 {
		return Done(nil)
	}
	return it.Bind(func(x interface{}) Iteratee {
		return Times_(n-1, it)
	})
}

func Validate(it Iteratee, pred func(interface{}) bool) Iteratee {
	return it.Bind(func(x interface{}) Iteratee {
		if pred(x) {
			return Done(x)
		} else {
			return Fail(NoMatch{"Validate"})
		}
	})
}
