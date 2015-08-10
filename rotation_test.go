package tailer_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomaso-mirodin/tailer"
)

// This test simply checks to make sure that we still work following a file rotaiton to what it does is writes out 100 bytes, and rotates the file half way through.
//The success condition for this test is just getting back more than 50 bytes, because that means we got past the rotation
func TestRmTouch(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		follow, err := tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

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
					require.NoError(os.Remove(filename), "couldn't delete file %q: %v", filename, err)
					t.Log("File removed")

					file, err = os.Create(filename)
					require.NoError(err, "failed to create new test file: %v", err)
					t.Log("File created again", file)
				}

				block := want[i : i+step]
				t.Logf("Writing block{%v} to file(%v)\n", block, file)
				file.Write(block)

				file.Sync()
				time.Sleep(time.Millisecond * time.Duration(step))
			}
			t.Log("Finished writing out all the bytes")
		}()

		got := make([]byte, len(want))
		_, err = io.ReadAtLeast(follow, got, 51)
		assert.NoError(err)

		t.Log("Got:", got)

		return nil
	})
}

// This test is a copy&paste of `TestRmTouch` but swapping the rm, for truncation
func TestMvTouch(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		follow, err := tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

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
					// CHANGES FROM `TestRmTouch` START HERE
					t.Log("Moving the old file out of the way", file)
					require.NoError(os.Rename(filename, filename+".old"), "Unable to rename the file")

					file, err = os.Create(filename)
					require.NoError(err, "failed to create new test file: %v", err)
					t.Log("File created again", file)
					// CHANGES FROM `TestRmTouch` STOP HERE
				}

				block := want[i : i+step]
				t.Logf("Writing block{%v} to file(%v)\n", block, file)
				file.Write(block)

				file.Sync()
				time.Sleep(time.Millisecond * time.Duration(step))
			}
			t.Log("Finished writing out all the bytes")
		}()

		got := make([]byte, len(want))
		_, err = io.ReadAtLeast(follow, got, 51)
		assert.NoError(err)

		t.Log("Got:", got)

		return nil
	})
}

// This test is a copy&paste of `TestRmTouch` but swapping the rm, for truncation
func TestTruncation(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	withTempFile(t, time.Millisecond*200, func(t *testing.T, filename string, file *os.File) error {
		follow, err := tailer.NewFile(filename)
		require.NoError(err, "Failed to create tailer.File")

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
					// CHANGES FROM `TestRmTouch` START HERE
					t.Log("truncating the file")
					file, err := os.OpenFile(filename, os.O_TRUNC, os.ModeTemporary)
					require.NoError(err, "unable to truncate file: %v", err)
					require.NoError(file.Close(), "failed to close the truncated file")
					t.Log("file truncated:", file)
					// CHANGES FROM `TestRmTouch` STOP HERE
				}

				block := want[i : i+step]
				t.Logf("Writing block{%v} to file(%v)\n", block, file)
				file.Write(block)

				file.Sync()
				time.Sleep(time.Millisecond * time.Duration(step))
			}
			t.Log("Finished writing out all the bytes")
		}()

		got := make([]byte, len(want))
		_, err = io.ReadAtLeast(follow, got, 51)
		assert.NoError(err)

		t.Log("Got:", got)

		return nil
	})
}
