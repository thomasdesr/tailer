package tailer

import "os"

// FileConfig lets you change the tailer.File, their primary use is as
// arguments to the tailer.File constructor, although they could be used later
type FileConfig func(*File) error

// Reset the tailer.File to read from the start of the file instead of its
// default of reading from the end
func ReadFromStart(t *File) error {
	_, err := t.file.Seek(0, os.SEEK_SET)
	return err
}
