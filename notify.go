package tailer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/fsnotify.v1"
)

func (t *File) notifyOnChanges() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := w.Add(filepath.Dir(t.filename)); err != nil {
		// If we can't watch the directory, we'll have to watch the file directly
		if err := w.Add(t.filename); err != nil {
			return err
		}

		// The problem is that now that we're whatching the file itself and not the directory it sits in, so filename changes don't get picked up. This means we have to poll the file path to detect when the file has rotated so we can re-watch the file.
		oldFile, err := t.file.Stat()
		if err != nil {
			return err
		}
		go func() {
			for !t.closed {
				newFile, err := os.Stat(t.filename)
				switch err {
				case nil:
					if !os.SameFile(oldFile, newFile) {
						if err := w.Add(t.filename); err != nil {
							t.errc <- err
						}
						if err := t.reopenFile(); err != nil {
							t.errc <- err
						}
						oldFile = newFile
					}
				default:
					// File missing, do nothing!
				}
				time.Sleep(PollIntervalSlow)
			}
		}()
	}

	go func(w *fsnotify.Watcher) {
		for !t.closed {
			select {
			case ev, open := <-w.Events:
				if !open {
					return
				}
				if pathEqual(ev.Name, t.filename) {
					err := t.handleFileEvent(ev)
					if err != nil {
						t.errc <- err
					}
				}
			case err, open := <-w.Errors:
				if !open {
					return
				}
				if err != nil {
					t.errc <- err
				}
			}
		}
	}(w)

	return nil
}

func (t *File) handleFileEvent(ev fsnotify.Event) error {
	switch {
	case isOp(ev, fsnotify.Create):
		// new file created with the same name, open it!
		return t.reopenFile()
	case isOp(ev, fsnotify.Write):
		// On write, check to see if the file has been truncated if not, fill the buffer
		if t.checkForTruncate() {
			if err := t.reopenFile(); err != nil {
				t.errc <- err
			}
		}
		return t.fill()
	case isOp(ev, fsnotify.Remove), isOp(ev, fsnotify.Rename):
		// wait for a new file to be created
		return nil
	case isOp(ev, fsnotify.Chmod):
		// Modified time on the file changed, noop
		return nil
	default:
		panic(fmt.Sprintf("received unknown fsnotify event: %#v", ev))
	}
}

func isOp(ev fsnotify.Event, op fsnotify.Op) bool {
	return ev.Op&op == op
}

func pathEqual(lhs, rhs string) bool {
	var err error
	lhs, err = filepath.Abs(lhs)
	if err != nil {
		return false
	}
	rhs, err = filepath.Abs(rhs)
	if err != nil {
		return false
	}
	return lhs == rhs
}
