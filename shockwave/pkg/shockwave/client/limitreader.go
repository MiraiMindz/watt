package client

import (
	"io"
	"sync"
)

// limitedReader wraps an io.Reader to limit how many bytes can be read.
// This is a pooled version of io.LimitReader to avoid allocations.
type limitedReader struct {
	R io.Reader // underlying reader
	N int64     // max bytes remaining
}

// limitedReaderPool pools limitedReader instances
var limitedReaderPool = sync.Pool{
	New: func() interface{} {
		return &limitedReader{}
	},
}

// getLimitedReader gets a pooled limitedReader
func getLimitedReader(r io.Reader, n int64) *limitedReader {
	lr := limitedReaderPool.Get().(*limitedReader)
	lr.R = r
	lr.N = n
	return lr
}

// putLimitedReader returns a limitedReader to the pool
func putLimitedReader(lr *limitedReader) {
	if lr != nil {
		lr.R = nil
		lr.N = 0
		limitedReaderPool.Put(lr)
	}
}

// Read reads from the underlying reader, respecting the limit
func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}
