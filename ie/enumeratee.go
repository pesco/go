package ie

import (
	"bytes"

	"github.com/pesco/go/monad"
)


type Enumeratee func(Iteratee) Iteratee
	// the returned Iteratee yields an Iteratee

// conceptually, Enumeratee is an Enumerator, but they are not convertible
// directly because monad.Monad has a different run-time representation than
// Iteratee. :/

// convert an Enumeratee to Enumerator. note the trivial implementation;
// Iteratee is converted to monad.Monad implicitly.
func (e Enumeratee) ToEnumerator() Enumerator {
	return func(it Iteratee) monad.Monad {
		return e(it)
	}
}

// enumeratee that passes its input as-is to the inner iteratee
var Pass Enumeratee = pass
func pass(it Iteratee) Iteratee {
	return Cont(func(s Stream) (Iteratee, Stream) {
		it, s := it.Feed(s)
		if it.k == nil || it.err != nil {
			return Done(it), s
		}
		return pass(it), s
	})
}

// attach an enumeratee to the output of an enumerator
func (e Enumerator) Pipe(ee Enumeratee) Enumerator {
	return func(inner Iteratee) monad.Monad {
		outer := ee(inner)
		return e(outer.Fuse())
	}
}

// when outer returns an Iteratee inner, outer.Fuse() is an iteratee that
// returns inner's result. when outer finishes, inner receives end of input.
func (outer Iteratee) Fuse() Iteratee {
	return outer.Bind(fuse)
}

func fuse(inner_ interface{}) Iteratee {
	inner := inner_.(Iteratee)
	if inner.err != nil {
		return inner
	}
	if inner.k != nil {
		inner, _ = inner.k(End)
	}
	return inner
}

func Prefix(it Iteratee, ee Enumeratee) Enumeratee {
	return func(inner Iteratee) Iteratee {
		return it.Then(ee(inner))
	}
}

func BreakAfter(sep []byte) Enumeratee {
	return func(inner Iteratee) (this Iteratee) {
		this = Cont(func(s Stream) (Iteratee, Stream) {
			if s == End {
				inner, _ = inner.Feed(s)
				return Done(inner), s
			}
			bs := s.Slice().([]byte)
			idx := bytes.Index(bs, sep)
			if idx == -1 {
				inner, _ = inner.Feed(s)
				return this, Empty
			}
			idx += len(sep)
			inner, _ := inner.Feed(Chunk(bs[:idx]))
			return Done(inner), Chunk(bs[idx:])
		})
		return
	}
}

// enumeratee equivalent of Many
func Repeat(a Enumeratee) (this Enumeratee) {
	this = func(it Iteratee) Iteratee {
		f := func(it_ interface{}) Iteratee {
			return this(it_.(Iteratee))
		}
		return OChoice(a(it).Bind(f), Done(it))
	}
	return
}

func Repeat1(a Enumeratee) Enumerator {	// XXX it's an enumeratee
	return a.ToEnumerator().Append(Repeat(a).ToEnumerator())
}

