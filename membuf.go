package membuf

import (
	"fmt"
	"io"

	"github.com/skillian/errors"
)

const (
	// pagePow2 is the only const that can be tweaked in this
	// implementation.
	pagePow2 = 15

	// pageSize is 2 to the (pagePow2)
	pageSize = 1 << pagePow2

	// pageMask is the page size minus one.  Go requires 2's complement, so
	// pageSize - 1 should work?
	pageMask = pageSize - 1
)

var (
	errInvalidWhence  = errors.New("invalid seek origin")
	errSeekOutOfRange = errors.New("seek out of range")
)

type page [pageSize]byte

// Buffer is like a bytes.Buffer but keeps memory around so it supports Seeking.
type Buffer struct {
	// pages is a slice of pages in the buffer
	pages []*page

	// pagei is the index into the buffer that has been seeked to.
	pagei int

	// lasti is the index of the end of the buffer.
	lasti int
}

func (b *Buffer) String() string {
	return fmt.Sprintf(
		"(*Buffer){pages: [%d]*page, pagei: %d, lasti: %d}",
		len(b.pages), b.pagei, b.lasti)
}

// Close the buffer is a no-op.
func (b *Buffer) Close() error { return nil }

func (b *Buffer) logData(name string) {
	p := make([]byte, b.lasti)
	for i := range b.pages {
		i *= pageSize
		pg := getBufferIndex(i).getPage(b)
		copy(p[i:], pg)
	}
	logger.Debug2("%s: Data: %v", name, p)
}

// Read implements io.Reader
func (b *Buffer) Read(p []byte) (n int, err error) {
	defer b.logData("Read")
	for t := p; len(t) > 0; t = p[n:] {
		logger.Debug0(b.String())
		pg := getBufferIndex(b.pagei).getPage(b)
		if bytesZero(pg) {
			logger.Warn0("bytes are zero")
		}
		m := copy(t, pg)
		b.pagei += m
		n += m
		logger.Debug0(b.String())
		if len(pg) == 0 {
			err = io.EOF
			break
		}
	}
	return
}

// Write implements io.Writer.  It always succeeds unless there's a panic from
// running out of memory.
func (b *Buffer) Write(p []byte) (n int, err error) {
	defer b.logData("Write")
	var newPage *page
	for s := p; len(s) > 0; s = p[n:] {
		logger.Debug0(b.String())
		bi := getBufferIndex(b.pagei)
		var pg []byte
		if b.pagei > 0 {
			pg = (*b.pages[bi.pageIndex])[bi.byteIndex:]
			pg = pg[:cap(pg)]
		}
		if len(pg) == 0 {
			logger.Debug0("adding another page")
			newPage = new(page)
			b.pages = append(b.pages, newPage)
			pg = (*newPage)[:]
		}
		m := copy(pg, s)
		b.pagei += m
		n += m
		if b.pagei > b.lasti {
			b.lasti = b.pagei
		}
	}
	logger.Debug0(b.String())
	return
}

// Seek implements io.Seeker
func (b *Buffer) Seek(offset int64, whence int) (n int64, err error) {
	o := int(offset)
	var current int
	switch whence {
	case io.SeekCurrent:
		current = b.pagei + o
	case io.SeekStart:
		current = o
	case io.SeekEnd:
		current = b.lasti + o
	default:
		return 0, errInvalidWhence
	}
	err = b.setPageI(current)
	return int64(b.pagei), err
}

// getNextPage is like getCurrentPage but moves the page index past the
// returned page so the next call returns new data.
func (b *Buffer) getNextPage(i int) ([]byte, bool) {
	pg := getBufferIndex(i).getPage(b)
	if len(pg) == 0 {
		return nil, false
	}
	b.pagei += len(pg)
	return pg, true
}

func (b *Buffer) setPageI(o int) error {
	if o < 0 || o > b.lasti {
		return errors.ErrorfWithCause(
			errSeekOutOfRange,
			"cannot set offset to %v.  Requires 0 <= offset <= %v",
			o, b.lasti)
	}
	b.pagei = o
	return nil
}

// bufferIndex separates a single index integer into its page index and then
// to the index of the byte within the page.
type bufferIndex struct {
	// pageIndex holds the index of the page in the buffer.
	pageIndex int

	// byteIndex holds the index of the byte within its page.
	byteIndex int
}

// getBufferIndex creates a 2-int tuple of a buffer's scalar index value into
// the page index and then the offset within the page.
func getBufferIndex(i int) bufferIndex {
	return bufferIndex{i >> pagePow2, i & pageMask}
}

// getPage gets the page from the buffer that the index corresponds to and then
// gets a slice of the page's bytes starting at the inner-page index.
func (i bufferIndex) getPage(b *Buffer) []byte {
	if len(b.pages) == 0 {
		return nil
	}
	pagePtr := b.pages[i.pageIndex]
	pg := (*pagePtr)[i.byteIndex:]
	lasti := getBufferIndex(b.lasti)
	lastPage := lasti.pageIndex == i.pageIndex
	if lastPage {
		pg = pg[:lasti.byteIndex-i.byteIndex]
	}
	return pg
}

func (i bufferIndex) value() int {
	return ((i.pageIndex << 12) & ^pageMask) | (i.byteIndex & pageMask)
}
