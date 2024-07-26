package bfutils

import (
	"bytes"
	"io"
	"os"
	"regexp"
)

type GlobalsType map[string]string

var Globals GlobalsType = make(GlobalsType)

func (g *GlobalsType) Set(key, value string) {
	(*g)[key] = value
}

func (g *GlobalsType) Get(key string) string {
	val, ok := (*g)[key]
	if !ok {
		return ""
	}
	return val
}

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

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			if v[i] == -1 || v[i+1] == -1 {
				groups = append(groups, "")
			} else {
				groups = append(groups, str[v[i]:v[i+1]])
			}
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
