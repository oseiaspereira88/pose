package pose

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var errGitBatchMissing = errors.New("git batch object is absent")

// gitBatchReader has one owner and one consumer, confined to an assessment.
// It reads raw blobs without filters, and never allocates beyond its caller's cap.
type gitBatchReader struct {
	root   string
	cmd    *exec.Cmd
	input  io.WriteCloser
	output *bufio.Reader
	cancel context.CancelFunc
	closed bool
}

func (r *gitBatchReader) start() error {
	if r.closed {
		return errors.New("git batch reader is closed")
	}
	if r.cmd != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cmd := exec.CommandContext(ctx, "git", "-C", r.root, "cat-file", "--batch")
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		_ = input.Close()
		cancel()
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = input.Close()
		_ = output.Close()
		cancel()
		return err
	}
	r.cmd, r.input, r.output, r.cancel = cmd, input, bufio.NewReader(output), cancel
	return nil
}

func (r *gitBatchReader) read(object string, max int) ([]byte, error) {
	if max < 0 || object == "" || strings.ContainsAny(object, "\r\n\x00") {
		return nil, errors.New("invalid Git batch request")
	}
	if err := r.start(); err != nil {
		return nil, err
	}
	fail := func(err error) ([]byte, error) { r.close(); return nil, err }
	if _, err := io.WriteString(r.input, object+"\n"); err != nil {
		return fail(err)
	}
	header, err := r.output.ReadSlice('\n')
	if err != nil {
		return fail(err)
	}
	line := strings.TrimSuffix(string(header), "\n")
	if line == object+" missing" {
		return nil, errGitBatchMissing
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[1] != "blob" {
		return fail(errors.New("invalid Git batch blob header"))
	}
	size, err := gitBatchBlobSize(fields[2], max)
	if err != nil {
		return fail(err)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r.output, body); err != nil {
		return fail(err)
	}
	separator, err := r.output.ReadByte()
	if err != nil || separator != '\n' {
		return fail(fmt.Errorf("invalid Git batch content delimiter: %v", err))
	}
	return body, nil
}

func gitBatchBlobSize(raw string, max int) (int, error) {
	size, err := strconv.Atoi(raw)
	if err != nil || size < 0 {
		return 0, errors.New("invalid Git batch blob size")
	}
	if size > max {
		return 0, errDesignDeltaTooLarge
	}
	return size, nil
}

func (r *gitBatchReader) close() {
	if r.closed {
		return
	}
	r.closed = true
	if r.input != nil {
		_ = r.input.Close()
	}
	if r.cancel != nil {
		r.cancel()
	}
	if r.cmd != nil {
		_ = r.cmd.Wait()
	}
}
