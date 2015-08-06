package tailer

import (
	"os"
	"time"
)

// Call this whenever we are going to need to reopen the `Tailer`'s file
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
		return err
	}

	return nil
}

// checkForTruncate stats the filename to see if the file has shrunk and therefore been truncated
// This isn't expected to handle IO errors, simply return True if the file has been truncated. (IO Errors may interfere with this happening)
// It also doesn't update the current size of the file (i.e. t.fileSize)
func (t *Tailer) checkForTruncate() bool {
	s, err := os.Stat(t.filename)

	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	if s.Size() < t.fileSize {
		// File size shrunk, that is our only sign for truncation
		return true
	}

	return false
}

// pollForRotations hits the filesystem every `pollIntervalSlow` looking to see if the file needs to be reopened
func (t *Tailer) pollForRotations() {
	previousFile, err := t.file.Stat()
	if err != nil {
		t.errc <- err
	}

	for {
		currentFile, err := os.Stat(t.filename)

		switch err {
		case nil:
			switch os.SameFile(currentFile, previousFile) {
			case true:
				if t.checkForTruncate() {
					if err := t.reopenFile(); err != nil {
						t.errc <- err
					}
				}
				t.fileSize = currentFile.Size()
			case false:
				previousFile = currentFile
				if err := t.reopenFile(); err != nil {
					t.errc <- err
				}
			}
		default:
			// Filename doens't seem to be there, wait for it to re-appear
		}

		time.Sleep(pollIntervalSlow)
	}
}

// func (t *Tailer) handleFileEvent(ev fsnotify.Event) fileAction {
// 	switch {
// 	case isOp(ev, fsnotify.Create):
// 		// new file created with the same name
// 		return reopenFile

// 	case isOp(ev, fsnotify.Write):
// 		// On write, check to see if the file has been truncated
// 		// If not, insure the bufio buffer is full
// 		switch f.checkForTruncate() {
// 		case true:
// 			return reopenFile
// 		case false:
// 			// COmeback and re-add this once we setup something other than polling for the fill
// 			return readFile
// 		}

// 	case isOp(ev, fsnotify.Remove), isOp(ev, fsnotify.Rename):
// 		// wait for a new file to be created
// 		return noop

// 	case isOp(ev, fsnotify.Chmod):
// 		// Modified time on the file changed, noop
// 		return noop

// 	default:
// 		panic(fmt.Sprintf("recieved unknown fsnotify event: %#v", ev))
// 	}
// }
