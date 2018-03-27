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
// The success condition for this test is just getting back more than 50 bytes, because that means we got past the rotation, not losing any bytes is a fundamental race condition in tailing files and can't ever be guaranteed.
func basicRotationTest(t *testing.T, rotationOperation func(t *testing.T, filename string, file *os.File)) {
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
					// Perform the file rotation operation
					rotationOperation(t, filename, file)
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

func TestRmTouch(t *testing.T) {
	rotationOperation := func(t *testing.T, filename string, file *os.File) {
		t.Log("Removing the file", file)
		require.NoError(t, os.Remove(filename), "couldn't delete file: %v", filename)
		t.Log("File removed")

		file, err := os.Create(filename)
		require.NoError(t, err, "failed to create new test file: %v", err)
		t.Log("File created again", file)
	}

	basicRotationTest(t, rotationOperation)
}

func TestMvTouch(t *testing.T) {
	rotationOperation := func(t *testing.T, filename string, file *os.File) {
		t.Log("Moving the old file out of the way", file)
		require.NoError(t, os.Rename(filename, filename+".old"), "Unable to rename the file")

		file, err := os.Create(filename)
		require.NoError(t, err, "failed to create new test file: %v", err)
		t.Log("File created again", file)
	}

	basicRotationTest(t, rotationOperation)
}

func TestTruncation(t *testing.T) {
	rotationOperation := func(t *testing.T, filename string, file *os.File) {
		t.Log("truncating the file")
		file, err := os.OpenFile(filename, os.O_TRUNC, os.ModeTemporary)
		require.NoError(t, err, "unable to truncate file: %v", err)
		require.NoError(t, file.Close(), "failed to close the truncated file")
		t.Log("file truncated:", file)
	}

	basicRotationTest(t, rotationOperation)
}
