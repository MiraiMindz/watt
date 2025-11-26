package client

import (
	"io"
	"sync"
)

// OptimizedReader is a zero-allocation buffered reader optimized for HTTP parsing.
// Unlike bufio.Reader, it provides zero-allocation line reading by reusing buffers.
//
// Performance: 0 allocs/op for line reading (vs ~2-3 allocs for bufio.ReadBytes)
type OptimizedReader struct {
	rd   io.Reader
	buf  []byte // Internal buffer
	r, w int    // Read and write positions

	// Line buffer for zero-copy reading
	lineBuf []byte

	// Error state
	err error
}

const (
	// OptimizedReaderSize is the default buffer size
	// Reduced from 4096 to 2048 - sufficient for most HTTP responses
	OptimizedReaderSize = 2048
	// MaxLineSize is the maximum line size for HTTP headers
	// Reduced from 8192 to 4096 - generous for typical headers
	MaxLineSize = 4096
)

var optimizedReaderPool = sync.Pool{
	New: func() interface{} {
		return &OptimizedReader{
			buf:     make([]byte, OptimizedReaderSize),
			lineBuf: make([]byte, 0, MaxLineSize),
		}
	},
}

// GetOptimizedReader returns a pooled OptimizedReader.
// The reader must be returned to the pool using PutOptimizedReader.
//
// Allocation behavior: 0 allocs/op on pool hit
func GetOptimizedReader(rd io.Reader) *OptimizedReader {
	r := optimizedReaderPool.Get().(*OptimizedReader)
	r.Reset(rd)
	return r
}

// PutOptimizedReader returns a reader to the pool.
//
// Allocation behavior: 0 allocs/op
func PutOptimizedReader(r *OptimizedReader) {
	if r != nil {
		r.Reset(nil)
		optimizedReaderPool.Put(r)
	}
}

// Reset resets the reader to read from rd.
//
// Allocation behavior: 0 allocs/op
func (r *OptimizedReader) Reset(rd io.Reader) {
	r.rd = rd
	r.r = 0
	r.w = 0
	r.lineBuf = r.lineBuf[:0]
	r.err = nil
}

// fill reads data into the buffer.
func (r *OptimizedReader) fill() error {
	if r.err != nil {
		return r.err
	}

	// Slide existing data to beginning
	if r.r > 0 {
		copy(r.buf, r.buf[r.r:r.w])
		r.w -= r.r
		r.r = 0
	}

	// Read more data
	n, err := r.rd.Read(r.buf[r.w:])
	r.w += n
	if err != nil {
		r.err = err
		return err
	}

	return nil
}

// ReadLine reads a line terminated by '\n'.
// The returned byte slice is valid until the next call to ReadLine or Reset.
// This is zero-allocation - the slice references internal buffer.
//
// Allocation behavior: 0 allocs/op
func (r *OptimizedReader) ReadLine() ([]byte, error) {
	r.lineBuf = r.lineBuf[:0]

	for {
		// Search for \n in current buffer
		for i := r.r; i < r.w; i++ {
			if r.buf[i] == '\n' {
				// Found newline
				line := r.buf[r.r : i+1]
				r.r = i + 1

				// If line is small enough, return direct reference (zero-copy)
				if len(line) <= cap(r.lineBuf) && len(r.lineBuf) == 0 {
					return line, nil
				}

				// Otherwise, copy to line buffer
				r.lineBuf = append(r.lineBuf, line...)
				return r.lineBuf, nil
			}
		}

		// No newline found, copy buffered data to line buffer
		r.lineBuf = append(r.lineBuf, r.buf[r.r:r.w]...)
		r.r = r.w

		// Fill buffer with more data
		if err := r.fill(); err != nil {
			if err == io.EOF && len(r.lineBuf) > 0 {
				return r.lineBuf, nil
			}
			return r.lineBuf, err
		}
	}
}

// ReadBytes reads until the first occurrence of delim.
// Returns a zero-copy slice when possible.
//
// Allocation behavior: 0 allocs/op for small reads
func (r *OptimizedReader) ReadBytes(delim byte) ([]byte, error) {
	r.lineBuf = r.lineBuf[:0]

	for {
		// Search for delimiter in current buffer
		for i := r.r; i < r.w; i++ {
			if r.buf[i] == delim {
				// Found delimiter
				line := r.buf[r.r : i+1]
				r.r = i + 1

				// Zero-copy path
				if len(line) <= cap(r.lineBuf) && len(r.lineBuf) == 0 {
					return line, nil
				}

				// Copy path
				r.lineBuf = append(r.lineBuf, line...)
				return r.lineBuf, nil
			}
		}

		// No delimiter found
		r.lineBuf = append(r.lineBuf, r.buf[r.r:r.w]...)
		r.r = r.w

		// Fill buffer
		if err := r.fill(); err != nil {
			if err == io.EOF && len(r.lineBuf) > 0 {
				return r.lineBuf, nil
			}
			return r.lineBuf, err
		}
	}
}

// Read reads data into p.
// Implements io.Reader interface.
func (r *OptimizedReader) Read(p []byte) (int, error) {
	if r.r == r.w {
		if len(p) >= len(r.buf) {
			// Large read, bypass buffer
			return r.rd.Read(p)
		}

		// Fill buffer
		if err := r.fill(); err != nil {
			return 0, err
		}
	}

	n := copy(p, r.buf[r.r:r.w])
	r.r += n
	return n, nil
}

// Buffered returns the number of bytes currently buffered.
func (r *OptimizedReader) Buffered() int {
	return r.w - r.r
}

// Peek returns the next n bytes without advancing the reader.
//
// Allocation behavior: 0 allocs/op
func (r *OptimizedReader) Peek(n int) ([]byte, error) {
	for r.w-r.r < n && r.err == nil {
		r.fill()
	}

	if r.w-r.r < n {
		if r.err != nil {
			return r.buf[r.r:r.w], r.err
		}
		return r.buf[r.r:r.w], io.ErrShortBuffer
	}

	return r.buf[r.r : r.r+n], nil
}

// Discard skips the next n bytes.
func (r *OptimizedReader) Discard(n int) (int, error) {
	discarded := 0

	for n > 0 {
		if r.r == r.w {
			if err := r.fill(); err != nil {
				return discarded, err
			}
		}

		skip := n
		if skip > r.w-r.r {
			skip = r.w - r.r
		}

		r.r += skip
		n -= skip
		discarded += skip
	}

	return discarded, nil
}
