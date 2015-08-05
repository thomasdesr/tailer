package tailer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/glycerine/rbuf"
	"gopkg.in/fsnotify.v1"
)

func readStuffs(r io.Reader) {
	b := make([]byte, 5)
	spew.Printf("%#v\n", r)
	spew.Println(r.Read(b))
	spew.Println(r.Read(b))
	spew.Println(r.Read(b))
}

type Tailer struct {
	filename string

	fmu     sync.Mutex
	file    *os.File
	watcher *fsnotify.Watcher

	ring *rbuf.FixedSizeRingBuf

	changes notifier

	errc chan error
}

type notifier struct {
	buffer chan struct{}
	disk   chan struct{}
}

func NewTailer(filename string) (*Tailer, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	p, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	t := &Tailer{
		filename: p,

		file:    f,
		watcher: w,

		ring: rbuf.NewFixedSizeRingBuf(4096),

		changes: notifier{
			buffer: make(chan interface{}),
			disk:   make(chan interface{}),
		},

		errc: make(chan error),
	}

	if err := w.Add(filepath.Dir(p)); err != nil {
		// If we can't watch the directory, we need to poll the file to see if it changes
		return nil, fmt.Errorf("Sadness, can't watch the dir, implement pollForChanges at some point you newb")
		// go f.pollForChanges()
	}

	go keepFilled()
	go detectRotations()

	return t, nil
}

// Read as much data is available in the file into the ring buffer
func (t *Tailer) fill() {
	t.fmu.Lock()
	defer t.fmu.Unlock()
	n, err := io.Copy(t.ring, t.file)
	switch err {
	case nil, io.ErrShortWrite, io.EOF:
		// Do nothing
	default:
		t.errc <- err
	}
}

func (t *Tailer) keepFilled() {
	for {
		select {
		case <-t.changes.disk:
			t.fill()
		case <-time.After(time.Second):
			t.fill()
		}
	}
}

func main() {
	t, err := NewTailer("garbage")

	fmt.Println(t, err)
	spew.Dump(t)
}
