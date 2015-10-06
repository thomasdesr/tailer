package tailer

import (
	"os"

	"github.com/glycerine/rbuf"
)

// FileConfig lets you change tailer.File objects, their primary use is as arguments to tailer.NewFile constructor, although they could be used later, but I don't suggest it
type FileConfig func(*File) error

// ReadFromStart will set the tailer.File to read from the start of the file instead of its default of reading from the end
func ReadFromStart() FileConfig {
	return func(t *File) error {
		_, err := t.file.Seek(0, os.SEEK_SET)
		return err
	}
}

// SetBufferSize sets the size of the internal ring buffer that tailers use to buffer reads and writes from disk
func SetBufferSize(i int) FileConfig {
	return func(t *File) error {
		t.ring = rbuf.NewFixedSizeRingBuf(i)
		return nil
	}
}

// PollForChanges will cause tailer to poll the file every PollIntervalFast for writes and PollIntervalSlow for rotations.
func PollForChanges() FileConfig {
	return func(t *File) error {
		t.rotationStrat = "polling"
		// go t.pollForUpdates(PollIntervalFast)
		// go t.pollForRotations(PollIntervalSlow)
		return nil
	}
}

// NotifyOnChanges will cause Tailer to use the filesystem's notification system (inotify, kqueue, etc) to detect file rotations as well as when the file is written to. It will attempt to listen to all events for the directory the file is in (this is required in order to detect when the file is recreated), if it cannot it will listen directly to the file for write updates, and poll to detect when a file is rotated
//
// Heads up, this is still a little flakey in terms of tests (from 0 -> 5% failure rate). I haven't seen any real world issues, but you may run into issues here. If you can get a reproduceable test case and send it to me that would be awesome :D
func NotifyOnChanges() FileConfig {
	return func(t *File) error {
		t.rotationStrat = "notify"
		// return t.notifyOnChanges()
		return nil
	}
}
