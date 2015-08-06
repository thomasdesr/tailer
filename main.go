package tailer

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glycerine/rbuf"
)

const (
	pollIntervalFast time.Duration = time.Millisecond * 15
	pollIntervalSlow time.Duration = time.Millisecond * 150
)

// Tailer needs a description
//
// TODO: Abstract changes/fills/rotations from polling or event based. I.e. have
// a rotate function that waits for a message on a rotateNow channel, have a
// fill buffer function that just waits for messages on the fillBufferNOw
// channel, etc. This way the choice of polling vs event based is handled
// purely by the args  (probably have them) spin up a goroutine that feeds those channels
type Tailer struct {
	filename string
	file     *os.File
	fileSize int64
	fmu      sync.Mutex

	ring *rbuf.FixedSizeRingBuf

	closed bool

	errc chan error
}

// NewTailer returns a new Tailer for the given file with the given Config
// options
func NewTailer(filename string, opts ...TailerConfig) (*Tailer, error) {
	var (
		path string
		f    *os.File
		err  error
	)

	if path, err = filepath.Abs(filename); err != nil {
		return nil, err
	}

	if f, err = os.OpenFile(filename, os.O_RDONLY, 0); err != nil {
		return nil, err
	}

	if _, err = f.Seek(0, os.SEEK_END); err != nil {
		_ = f.Close()
		return nil, err
	}

	t := &Tailer{
		filename: path,
		file:     f,

		ring: rbuf.NewFixedSizeRingBuf(4096),

		errc: make(chan error),
	}

	for _, opt := range opts {
		err := opt(t)
		if err != nil {
			return nil, err
		}
	}

	// Use polling or event based change detection / rotation
	switch {
	default:
		go t.pollForChanges(pollIntervalFast)
		go t.pollForRotations(pollIntervalSlow)
	}

	return t, nil
}

// Read is the implementation of the io.Reader interface below are the
// implemenation details
//
// Read will return (0, io.EOF) to any call after the Reader is closed.
//
// Future Note: This is not set in stone, I am torn between allowing the current
// buffer to be flushed by Read after Close() is called and its current
// behavior. However I have taken the conservative route and currently
// EOF all post-close writes.
func (t *Tailer) Read(b []byte) (int, error) {
	// Don't return 0, nil
	for t.ring.Readable == 0 && !t.closed {
		time.Sleep(pollIntervalFast)
	}

	if t.closed == true {
		return 0, io.EOF
	}

	// Check for any waiting errors
	select {
	case err := <-t.errc:
		return 0, err
	default:
	}

	return t.ring.Read(b)
}

// Close is the implementation of the io.Closer interface with implemenation
//
// This closes the Tailer, which currently prevents any further reads from the
// tailer.
func (t *Tailer) Close() error {
	t.closed = true
	return t.file.Close()
}

// Turn this into an example at some point XD
// func main() {
// 	t, err := NewTailer("/tmp/garbage")
// 	fmt.Println(t, err)
// 	if err != nil {
// 		return
// 	}

// 	spew.Dump(t)

// 	time.Sleep(time.Second * 10)

// 	s := bufio.NewScanner(t)
// 	for s.Scan() {
// 		spew.Println(s.Text())
// 	}

// 	if err := s.Err(); err != nil {
// 		spew.Println("Error:", err)
// 	}

// 	spew.Dump(t)
// }
