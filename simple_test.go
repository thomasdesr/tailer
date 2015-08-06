package tailer_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/thomaso-mirodin/tailer"
)

func TestImpl(t *testing.T) {
	var tail io.ReadCloser
	var err error
	withTempFile(t, time.Millisecond*150, func(t *testing.T, filename string, file *os.File) error {
		t.Log("Creating Tailer")
		tail, err = tailer.NewTailer(filename)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Tailer created")

		t.Log("Closing Tailer")
		if err := tail.Close(); err != nil {
			t.Fatalf("failed to close tailer.Tailer: %v", err)
		}
		t.Log("Tailer closed")

		t.Log("Reading from Tailer after close")
		_, err = tail.Read(make([]byte, 1))
		t.Log("Reading from Tailer after closer returned err ->", err)
		switch err {
		case nil, io.EOF:
			return nil
		default:
			return err
		}
	})
}

func TestCanFollowFile(t *testing.T) {
	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		tail, err := tailer.NewTailer(filename)
		if err != nil {
			return fmt.Errorf("failed creating tailf.follower: '%v'", err)
		}

		go func() {
			for i := 0; i < 10; i++ {
				t.Logf("Writing: %v", i)
				file.Write([]byte{byte(i)})
				time.Sleep(time.Millisecond * 10)
			}
		}()

		t.Log("Preparing to read 10 bytes")
		_, err = io.ReadAtLeast(tail, make([]byte, 10), 10)
		t.Log("Read 10 bytes, err ->", err)
		return err
	})
}

// This test simply checks to make sure that we still work following a file rotaiton
// to what it does is writes out 100 bytes, and rotates the file half way through.
// The success condition for this test is just getting back more than 50 bytes, because
// that means we got past the rotation
func TestCanFollowFileOverwritten(t *testing.T) {
	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		follow, err := tailer.NewTailer(filename)
		if err != nil {
			t.Fatalf("creating Tailer: %v", err)
		}

		want := make([]byte, 100)
		for i := 0; i < 100; i++ {
			want[i] = byte(i)
		}

		t.Log("Want:", want)

		go func() {
			max := len(want)
			step := max / 10
			for i := 0; i < max; i += step {
				if i == max/2 {
					t.Log("Removing the file", file)
					if err := os.Remove(filename); err != nil {
						t.Fatalf("couldn't delete file %q: %v", filename, err)
					}
					t.Log("File removed")

					file, err = os.Create(filename)
					if err != nil {
						t.Fatalf("failed to create new test file: %v", err)
					}
					t.Log("File created again", file)
				}

				block := want[i : i+step]
				t.Logf("Writing block{%v} to file(%v)\n", block, file)
				_, err := file.Write(block)
				if err != nil {
					t.Fatalf("failed to write to test file: %v", err)
				}

				file.Sync()
				time.Sleep(time.Millisecond * time.Duration(step))
			}
			t.Log("Finished writing out all the bytes")
		}()

		got := make([]byte, len(want))
		n, err := io.ReadAtLeast(follow, got, 51)
		if err != nil {
			t.Error(n, err)
		}

		t.Log("Got:", got)

		return nil
	})
}

func withTempFile(t *testing.T, timeout time.Duration, action func(t *testing.T, filename string, file *os.File) error) {
	dir, err := ioutil.TempDir(os.TempDir(), "tailer")
	if err != nil {
		t.Fatalf("couldn't create temp dir: '%v'", err)
	}
	defer os.RemoveAll(dir)

	file, err := ioutil.TempFile(dir, "tailer_test")
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("couldn't create temp file: '%v'", err)
	}
	defer file.Close()

	errc := make(chan error)
	go func() { errc <- action(t, file.Name(), file) }()

	select {
	case err = <-errc:
		if err != nil {
			t.Errorf("failure: %v", err)
		}
	case <-time.After(timeout):
		t.Error("test took too long :(")
	}
}
