package tailer

import "os"

// FileConfig lets you change the tailer.File, their primary use is as arguments to the tailer.File constructor, although they could be used later
type FileConfig func(*File) error

// ReadFromStart will set the tailer.File to read from the start of the file instead of its default of reading from the end
func ReadFromStart(t *File) error {
	_, err := t.file.Seek(0, os.SEEK_SET)
	return err
}

// PollForChanges will cause tailer to poll the file every `pollIntervalFast` for writes and `pollIntervalSlow` for rotations.
func PollForChanges(t *File) error {
	t.rotationStrat = "polling"
	// go t.pollForUpdates(pollIntervalFast)
	// go t.pollForRotations(pollIntervalSlow)
	return nil
}

// NotifyOnChanges will cause Tailer to use fsnotify(inotify, kqueue, etc) to detect file rotations as well as when the file is written to. It will attempt to listen to all events for the directory the file is in (this is required in order to detect when the file is recreated), if it cannot it will listen directly to the file for write updates, and poll to detect when a file is rotated
//
// Heads up, this is still a little flakey in terms of tests (from 0 -> 5% failure rate). I haven't seen any real world issues, but ymmv
func NotifyOnChanges(t *File) error {
	t.rotationStrat = "notify"
	// return t.notifyOnChanges()
	return nil
}
