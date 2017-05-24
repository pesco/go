// Iteratees and Enumerators
package ie

import (
	"io"

	"github.com/pesco/go/monad"
)


// variant data type - fat struct representation
// valid field assignments:
//
//   1. result, nil, nil   -- done (result may be nil)
//   2. nil   , k,   nil   -- continuing
//   3. nil   , k,   err   -- recoverable error
//
type Iteratee struct {
	result  interface{}
	k       func(Stream) (Iteratee, Stream)
	err     error
}


// constructors...

func Done(x interface{}) Iteratee {
	return Iteratee{x, nil, nil}
}

func Cont(k func(Stream) (Iteratee, Stream)) Iteratee {
	return Iteratee{nil, k, nil}
}

func Stop(e error, k func(Stream) (Iteratee, Stream)) Iteratee {
	return Iteratee{nil, k, e}
}

func Fail(e error) Iteratee {
	k := func(s Stream) (Iteratee, Stream) {return Fail(e), s}
	return Stop(e, k)
}

func Raise(msg error) Iteratee {
	k := func(s Stream) (Iteratee, Stream) {return Done(nil), s}
	return Stop(msg, k)
}


// read-only field access...

func (it Iteratee) IsDone() bool {return it.k == nil}
func (it Iteratee) IsCont() bool {return it.k != nil && it.err == nil}
func (it Iteratee) IsStop() bool {return it.err != nil}

func (it Iteratee) Result() interface{}           {return it.result}
func (it Iteratee) Err() error                    {return it.err}
func (it Iteratee) K(s Stream) (Iteratee, Stream) {return it.k(s)}


// methods...

func (it Iteratee) Feed(s Stream) (Iteratee, Stream) {
	if it.k == nil || it.err != nil {	// if it is done or stopped on error
		return it, s					//   remain in error state
	}
	return it.k(s)
}

func (it Iteratee) Run() interface{} {
	it, _ = it.Feed(End)
	if it.k != nil {
		panic(it.err)
	}
	return it.result
}


// monad instance...

func (it Iteratee) Bind(f func(interface{}) Iteratee) Iteratee {
	if it.k == nil {
		return f(it.result)
	}
	k := func(s Stream) (Iteratee, Stream) {
			it, s := it.k(s)			// feed input to 'it' and
			if it.k != nil {			// if it stops,
				return it.Bind(f), s	// bind f to it again and return
			}
			// when 'it' is done, call f to continue and pass the rest of s
			return f(it.result).Feed(s)
		}
	return Iteratee{nil, k, it.err}
}

// a.Then(b) = a.Bind({return b})
func (a Iteratee) Then(b Iteratee) Iteratee {
	if a.k == nil {
		return b
	}
	k := func(s Stream) (Iteratee, Stream) {
			a, s := a.k(s)
			if a.k != nil {
				return a.Then(b), s
			}
			return b.Feed(s)
		}
	return Iteratee{nil, k, a.err}
}

// Return = Done
func (a Iteratee) ThenReturn(x interface{}) Iteratee {
	return a.Then(Done(x))
}

func (a Iteratee) ThenReturn_(x interface{}) monad.Monad {
	return a.ThenReturn(x)
}

func (it Iteratee) Bind_(f_ func(interface{}) monad.Monad) monad.Monad {
	f := func(x interface{}) Iteratee {return f_(x).(Iteratee)}
	return it.Bind(f)
}

func (a Iteratee) Then_(b_ monad.Monad) monad.Monad {
	return a.Then(b_.(Iteratee))
}


// primitive iteratees...

// consume and return the first element of the input
var Head Iteratee = Cont(k_head)
func k_head(s Stream) (Iteratee, Stream) {
	if s == End {
		return Fail(NoMatch{"end of input"}), s
	}
	if s == Empty {
		return Cont(k_head), s
	}
	x, s := s.Take1()
	return Done(x), s
}

// consume and discard the first n elements of the input
func Skip(n int) Iteratee {
	if n <= 0 {
		return Done(nil)
	}
	return Cont(func(s Stream) (Iteratee, Stream) {
		l := s.Len()
		if l < n {
			return Skip(n-l), Empty
		}
		return Done(nil), s.Drop(n)
	})
}

func Write(w io.Writer) (this Iteratee) {
	this = Cont(func(s Stream) (Iteratee, Stream) {
		if s == End {
			return Done(nil), s
		}
		bs := s.Slice().([]byte)
		n, err := w.Write(bs)
		if n > 0 {
			s = Chunk(bs[n:])
		}
		if err != nil {
			return Stop(err, this.k), s
		}
		return this, s
	})
	return
}
