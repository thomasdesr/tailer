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
	PollIntervalFast time.Duration = time.Millisecond * 15
	PollIntervalSlow time.Duration = time.Millisecond * 150
)

// TODO: (Mayyyybe) Abstract changes/fills/rotations from polling or event based. I.e. have a rotate function that waits for a message on a rotateNow channel, have a fill buffer function that just waits for messages on the fillBufferNow channel, etc. This way the way the logic around filling of buffers is abstracted away from the choice of which strategy to use (polling vs inotify).
//
// TODO: Have fill() read from a io.MultiReader instead of directly from the file, when a rotation is detected, create a new io.MultiReader from the old io.Reader and the new file. So something like this:
//
//		t.reader = io.NewMultiReader(t.reader, t.file)
//
// Where @ start time t.reader starts out as the file, but upon the first rotation is swapped out for a io.MultiReader which includes the old file and a the new file.

// File is the container for all the logic around tailing a single file
type File struct {
	filename string
	file     *os.File
	fileSize int64
	fmu      sync.Mutex

	rotationStrat string // Omghacky, get rid of me :(

	ring *rbuf.FixedSizeRingBuf

	closed bool

	errc chan error
}

// NewFile returns a new File for the given file with the given Config options
func NewFile(filename string, opts ...FileConfig) (*File, error) {
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

	t := &File{
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

	switch t.rotationStrat {
	case "notify":
		if err := t.notifyOnChanges(); err != nil {
			return nil, err
		}
	default:
		go t.pollForUpdates(PollIntervalFast)
		go t.pollForRotations(PollIntervalSlow)
	}

	return t, nil
}

// Read is the implementation of the io.Reader interface below are the implemenation details
//
// Read will return (0, io.EOF) to any call after the Reader is closed.
//
// Future Note: This is not set in stone, I am torn between allowing the current buffer to be flushed by Read after Close() is called and its current behavior. However I have taken the conservative route and currently EOF all post-close writes.
func (t *File) Read(b []byte) (int, error) {
	// Don't return 0, nil
	for t.ring.Readable == 0 && !t.closed {
		time.Sleep(PollIntervalFast) // Maybe swap this out for a notification at some point, but tbh, this works
	}

	if t.closed == true {
		return 0, io.EOF
	}

	// Check for any waiting errors
	select {
	case err := <-t.errc:
		if err != nil { // Just in case XD
			return 0, err
		}
	default:
	}

	return t.ring.Read(b)
}

// Close is the implementation of the io.Closer interface with implemenation
//
// This closes the File, which currently prevents any further reads from the tailer.
func (t *File) Close() error {
	t.closed = true
	return t.file.Close()
}

// Read as much data is available in the file into the ring buffer ignoring short writes (buffer is full), and EOFs (no more data to read from the disk) as they are expected
func (t *File) fill() error {
	t.fmu.Lock()
	_, err := io.Copy(t.ring, t.file)
	t.fmu.Unlock()
	switch err {
	case nil, io.ErrShortWrite, io.EOF:
		return nil
	default:
		return err
	}
}

// Call this whenever we are going to need to reopen the `Tailer`'s file
func (t *File) reopenFile() error {
	t.fmu.Lock()
	defer t.fmu.Unlock()

	if t.file != nil {
		if err := t.file.Close(); err != nil {
			return err
		}
	}

	var err error
	t.file, err = os.OpenFile(t.filename, os.O_RDONLY, 0)
	switch {
	case os.IsNotExist(err):
		t.file = nil
	default:
		return err
	}

	return nil
}

// checkForTruncate stats the filename to see if the file has shrunk and therefore been truncated
// This isn't expected to handle IO errors, simply return True if the file has been truncated. (IO Errors may interfere with this happening)
func (t *File) checkForTruncate() bool {
	s, err := os.Stat(t.filename)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	if s.Size() < t.fileSize {
		// File size shrunk, that is the sign for truncation
		t.fileSize = s.Size()
		return true
	}

	t.fileSize = s.Size()
	return false
}

// Turn this into an example at some point XD
// func main() {
// 	t, err := NewFile("/tmp/garbage")
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
