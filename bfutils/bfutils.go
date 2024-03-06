package bfutils

import (
	"bytes"
	"io"
	"os"
)

type Flusher interface {
	Flush() error
}

type FileOrMemReader interface {
	io.Reader
}
type FileOrMemWriter interface {
	io.Writer
	Flusher
}

type StdoutWrapper struct {
	*os.File
}

func (out StdoutWrapper) Flush() error {
	return out.Sync()
}

type BufferWrapper struct {
	*bytes.Buffer
}

func (out BufferWrapper) Flush() error {
	return nil
}

func (out BufferWrapper) Bytes() *bytes.Buffer {
	return out.Bytes()
}

func WrapStdout(out *os.File) StdoutWrapper {
	return StdoutWrapper{out}
}

func WrapBuffer(buf *bytes.Buffer) FileOrMemWriter {
	return BufferWrapper{buf}
}
