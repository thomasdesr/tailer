package tailer

import (
	"io"
	"time"
)

// Read as much data is available in the file into the ring buffer
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

func (t *File) pollForChanges(d time.Duration) {
	for !t.closed {
		if err := t.fill(); err != nil {
			if err = t.reopenFile(); err != nil {
				t.errc <- err
			}
		}

		time.Sleep(d)
	}
}

// func (t *File) notifyForChanges() {
// 	// tbd
// }
