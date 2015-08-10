package tailer_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomaso-mirodin/tailer"
)

// Make sure this adheres to a io.ReadCloser & check that reads after closes return (0, io.EOF)
func TestBasicImpl(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var (
		tail io.ReadCloser
		n    int
		err  error
	)
	withTempFile(t, time.Millisecond*150, func(t *testing.T, filename string, file *os.File) error {
		tail, err = tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

		err = tail.Close()
		require.NoError(err, "Failed to close tailer.File")

		n, err = tail.Read(make([]byte, 1))

		assert.Equal(0, n, "Calls to Read after Close is called should return 0 bytes read")
		assert.Equal(io.EOF, err, "Calls to Read after Close is called should return io.EOF as their error")

		return nil
	})
}

// Can we do the most basic thing, can we follow writes to the file overtime
func TestCanFollowFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		tail, err := tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

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
		assert.NoError(err, "Read shouldn't return an error here")
		return err
	})
}

// Run for 50ms constantly trying to read from something that has nothing to read
// This is pretty much here to make sure we don't let our Reader "spin" i.e. return (0, nil)
func TestSpinningReader(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	withTempFile(t, time.Millisecond*150, func(t *testing.T, filename string, file *os.File) error {
		tail, err := tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

		// Touch the file repeatedly
		go func() {
			for _ = range time.Tick(time.Millisecond * 5) {
				t.Log("Touching the file")
				os.Chtimes(filename, time.Now(), time.Now())
			}
		}()

		// Read from the file as quickly as possible
		var readCount int
		go func() {
			buf := make([]byte, 1000)
			for stop := time.After(time.Millisecond * 50); ; {
				select {
				case <-stop:
					return
				default:
					t.Log("Read called")
					_, err := tail.Read(buf)
					assert.NoError(err, "read error: %v", err)
					t.Log("Read completed")
					readCount++
					if readCount > 5 {
						assert.Fail("Spinning on read")
					}
				}
			}
		}()

		// This sleep & the go func() of the code above is there because Read should deadlock the thread it is running in
		time.Sleep(time.Millisecond * 50)

		t.Logf("Reader read '%v' times", readCount)
		return nil
	})
}

// TODO: Despite defering here, we still aren't quite cleaning up the files all the time :(
func withTempFile(t *testing.T, timeout time.Duration, action func(t *testing.T, filename string, file *os.File) error) {
	dir, err := ioutil.TempDir(os.TempDir(), "tailer")
	require.NoError(t, err, "couldn't create temp dir: '%v'", err)
	defer os.RemoveAll(dir)

	file, err := ioutil.TempFile(dir, "tailer_test")
	require.NoError(t, err, "couldn't create temp file: '%v'", err)
	defer file.Close()

	errc := make(chan error)
	go func() { errc <- action(t, file.Name(), file) }()

	select {
	case err = <-errc:
		assert.NoError(t, err)
	case <-time.After(timeout):
		assert.Fail(t, "test took too long :(")
	}
}
