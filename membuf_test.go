package membuf_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/skillian/logging"

	"github.com/skillian/membuf"
)

const helloWorld = "Hello, World!"

type op int

const (
	read op = iota
	write
)

type action struct {
	// read or write operation
	op

	// offset seeked to before performing op.
	offset int64
	// whence seeked to before performing op.
	whence int

	// when op is read, the expected bytes to read.  When op is write,
	// the bytes to write.
	buf []byte

	// err is an expected error.
	err error
}

type bufferTest struct {
	name    string
	actions []action
}

var (
	helloWorldBytes = []byte(helloWorld)

	bufferTests = [...]bufferTest{
		bufferTest{
			name: "rwHelloWorld",
			actions: []action{
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{read, 0, io.SeekStart, helloWorldBytes, io.EOF},
			},
		},
		bufferTest{
			name: "rwrwHelloWorld",
			actions: []action{
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{read, 0, io.SeekStart, helloWorldBytes, io.EOF},
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{read, 0, io.SeekStart, helloWorldBytes, io.EOF},
			},
		},
		bufferTest{
			name: "wwwHelloWorldRRRHelloWorld",
			actions: []action{
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{write, 0, io.SeekCurrent, helloWorldBytes, nil},
				action{read, 0, io.SeekStart, []byte(helloWorld + helloWorld + helloWorld), io.EOF},
			},
		},
	}
)

func TestBuffer(t *testing.T) {
	for _, bt := range bufferTests {
		t.Run(bt.name, func(t *testing.T) {
			b := new(membuf.Buffer)
			for _, a := range bt.actions {
				_, err := b.Seek(a.offset, a.whence)
				if err != nil {
					t.Fatalf("%T.Seek(%v, %v) %v", b, a.offset, a.whence, err)
				}
				switch a.op {
				case read:
					p := make([]byte, len(a.buf))
					n, err := b.Read(p)
					if err != nil && err != io.EOF {
						t.Fatal("error from read:", err)
					}
					if n != len(p) {
						t.Fatalf("read %d bytes", n)
					}
					if !bytes.Equal(p, a.buf) {
						t.Fatal("read expected:", a.buf, "but got", p)
					}
				case write:
					n, err := b.Write(a.buf)
					if err != nil {
						t.Fatal("error from write:", err)
					}
					if n != len(a.buf) {
						t.Fatalf("wrote %d bytes", n)
					}
				default:
					panic("invalid op")
				}
			}
		})
	}
}

func init() {
	logger := logging.GetLogger("membuf")
	h := new(logging.ConsoleHandler)
	h.SetFormatter(logging.FormatterFunc(format))
	h.SetLevel(logging.DebugLevel)
	logger.SetLevel(logging.DebugLevel)
	logger.AddHandler(h)
}

func format(e *logging.Event) string {
	return strings.Join([]string{
		e.FuncName,
		": ",
		fmt.Sprintf(e.Msg, e.Args...),
		"\n",
	}, "")
}
