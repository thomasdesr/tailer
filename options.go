package tailer

import "os"

type TailerConfig func(*Tailer) error

func ReadFromStart(t *Tailer) error {
	_, err := t.file.Seek(0, os.SEEK_SET)
	return err
}
