package tailer

import (
	"os"
	"time"
)

// pollForUpdates just sits here and tries to refill the buffer every interval
func (t *File) pollForUpdates(d time.Duration) {
	for !t.closed {
		if err := t.fill(); err != nil {
			if err = t.reopenFile(); err != nil {
				t.errc <- err
			}
		}

		time.Sleep(d)
	}
}

// pollForRotations hits the filesystem every interval looking to see if the file at `filename` path is different from what it was previously
func (t *File) pollForRotations(d time.Duration) {
	previousFile, err := t.file.Stat()
	if err != nil {
		t.errc <- err
	}

	for !t.closed {
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
			case false:
				previousFile = currentFile
				if err := t.reopenFile(); err != nil {
					t.errc <- err
				}
			}
		default:
			// Filename doens't seem to be there (or something else weird), wait for it to re-appear
		}

		time.Sleep(d)
	}
}
