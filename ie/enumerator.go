package ie

import (
	"io"
	"fmt"

	"github.com/pesco/go/monad"
)

type Enumerator func(Iteratee) monad.Monad
	// the returned Monad yields an Iteratee

func EnumString(s string) Enumerator {
	return func(it Iteratee) monad.Monad {
		return monad.IO(func() interface{} {
			it, _ = it.Feed(Chunk([]byte(s)))
			return it
		})
	}
}

// stop an iteratee with this message to signal a supporting enumerator
// to seek to the given position in the stream, e.g. a file. a negative offset
// counts from the end of the file, where -1 indicates the last byte.
// to seek relative to the current position, use SeekRel.
type Seek struct {
	Offset int64
}
func (sk Seek) Error() string {
	return fmt.Sprintf("tried to seek (to position %#x)", sk.Offset)
}
type SeekRel struct {
	Offset int64
}
func (sk SeekRel) Error() string {
	return fmt.Sprintf("tried to seek (by %v bytes)", sk.Offset)
}

func Read(r io.Reader) Enumerator {
	return read(r, false)
}
func SeekableRead(r io.ReadSeeker) Enumerator {
	return read(r, true)
}
func read(r io.Reader, seekable bool) Enumerator {
	buf := make([]byte, 1024)

	return func(it Iteratee) monad.Monad {
		return monad.IO(func() interface{} {
			for it.k != nil {
				if it.err != nil {
					// Iteratee has stopped - end here or seek if supported
					seek := false
					where := int64(0)
					whence := io.SeekStart

					switch sk := it.err.(type) {
					case Seek:
						seek = true
						where = sk.Offset
						if where < 0 {
							whence = io.SeekEnd
						}
					case SeekRel:
						seek = true
						where = sk.Offset
						whence = io.SeekCurrent
					}

					if seek && seekable {
						_, err := r.(io.ReadSeeker).Seek(where, whence)
						if err != nil {
							it, _ = it.Feed(End)	// XXX it.Feed(End(err))
							return it
						}
					} else {
						return it
					}
				}

				n, err := r.Read(buf)
				if n > 0 {
					// process any bytes returned, regardless of errors
					// cf. https://golang.org/pkg/io/#Reader
					it, _ = it.k(Chunk(buf[0:n]))
				}
				if err != nil {
					if err != io.EOF {
						it, _ = it.Feed(End)	// XXX it.Feed(End(err))
					}
					// NB: end-of-file does not feed End, iteratee can go on
					//     with another enumerator!
					return it
				}
			}
			return it
		})
	}
}

// run two enumerators after another
func (a Enumerator) Append(b Enumerator) Enumerator {
	return func(it Iteratee) monad.Monad {
		f := func(it_ interface{}) monad.Monad {
			return b(it_.(Iteratee))
		}
		return a(it).Bind_(f)
	}
}
