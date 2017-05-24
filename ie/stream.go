package ie

import (
	"reflect"
)


type Endianness int
const (
	LE Endianness = iota
	BE
)

// stores a chunk of elements of the same (but arbitrary) type
// XXX also store optional error on End
type Stream struct {
	slice interface{}
	isEnd bool

	isBit bool
	bitorder Endianness
	offset uint8	// for bit streams
}


// constructors...

var End   Stream = Stream{nil, true, false, LE, 0}
var Empty Stream = Stream{nil, false, false, LE, 0}

func Chunk(slice interface{}) Stream {
	v := reflect.ValueOf(slice)
	if v.Kind() == reflect.String {
		slice = v.Convert(t_bytes).Bytes()
	} else if v.Kind() != reflect.Slice {
		panic("Chunk(): slice type expected")
	}
	if v.Len() == 0 {
		return Empty
	}
	return Stream{slice, false, false, LE, 0}
}
var t_bytes reflect.Type = reflect.TypeOf([]byte(nil))

func BitChunk(slice []byte, bitorder Endianness, offset uint8) Stream {
	if len(slice) == 0 {
		return Empty
	}
	return Stream{slice, false, true, bitorder, offset}
}


// accesors...

func (s *Stream) Slice() interface{} {
	if s.isBit {
		panic("Slice() called on bitstream")
	}
	if s.isEnd {
		panic("Slice() called on End")
	}
	return s.slice
}

func (s *Stream) Len() int {
	if s.slice == nil {
		return 0
	} else {
		return reflect.ValueOf(s.slice).Len()
	}
}

func (s *Stream) Bytes() []byte {
	if !s.isBit {
		panic("Bytes() called on non-bitstream")
	}
	if s.isEnd {
		panic("Bytes() called on End")
	}
	return s.slice.([]byte)
}

func (s *Stream) Offset() uint8 {
	return s.offset
}

func (s *Stream) Endian() Endianness {
	return s.bitorder
}


// primitives...

func (s *Stream) Drop(n int) Stream {
	v := reflect.ValueOf(s.slice)
	return Chunk(v.Slice(n,v.Len()).Interface())
}

func (s *Stream) Take1() (interface{}, Stream) {
	var x interface{}
	if s.isBit {
		bs := s.Bytes()
		switch s.bitorder {
		case LE: x = (bs[0] >> s.offset) & 1
		case BE: x = (bs[0] >> (7-s.offset)) & 1
		}
		if s.offset >= 7 {
			return x, BitChunk(bs[1:], s.bitorder, 0)
		}
		return x, BitChunk(bs, s.bitorder, s.offset+1)
	}
	v := reflect.ValueOf(s.slice)
	x = v.Index(0).Interface()
	return x, Chunk(v.Slice(1,v.Len()).Interface())
}
