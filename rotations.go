package tailer

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/fsnotify.v1"
)

func (t *Tailer) detectRotations() {
	if err := t.watcher.Add(filepath.Dir(path)); err != nil {
		// If we can't watch the directory, we need to poll the file to see if it changes
		// go f.pollForChanges()
	}
	defer t.watcher.Close()

	// Detect Move&Touch

	// Detect Remove&Touch

	// Detect CopyTruncate

}

func (t *Tailer) watchFile() (rotateNow chan struct{}, err error) {
	for {
		select {
		case ev, open := <-t.watcher.Events:
			if !open {
				return
			}
			if pathEqual(ev.Name, t.filename) {
				err := f.handleFileEvent(ev)
				if err != nil {
					f.errc <- err
					return
				}
			}
		case err, open := <-t.watcher.Errors:
			if !open {
				return
			}
			if err != nil {
				f.errc <- err
			}
		}
	}
}

func (f *follower) handleFileEvent(ev fsnotify.Event) error {
	switch {
	case isOp(ev, fsnotify.Create):
		// new file created with the same name
		return f.reopenFile()

	case isOp(ev, fsnotify.Write):
		// On write, check to see if the file has been truncated
		// If not, insure the bufio buffer is full
		switch f.checkForTruncate() {
		case nil:
			return f.fillFileBuffer()
		case ErrFileRemoved{}:
			// If file was written to and then removed before we could even Stat the file, just wait for the next creation
			return nil
		default:
			return f.reopenFile()
		}

	case isOp(ev, fsnotify.Remove), isOp(ev, fsnotify.Rename):
		// wait for a new file to be created
		return nil

	case isOp(ev, fsnotify.Chmod):
		// Modified time on the file changed, noop
		return nil

	default:
		return fmt.Errorf("recieved unknown fsnotify event: %#v", ev)
	}
}

func (t *Tailer) reopenFile() error {
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
		t.errc <- err
	}

	return nil
}

func isOp(ev fsnotify.Event, op fsnotify.Op) bool {
	return ev.Op&op == op
}
