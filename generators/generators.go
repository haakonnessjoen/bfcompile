package generators

import (
	l "bcomp/lexer"
	"fmt"
	"os"
	"strings"
)

var PrintWarnings = true

type ParseToken struct {
	Pos    l.Position
	Tok    l.Token
	Extra  int
	Extra2 int
}

func (t ParseToken) String() string {
	return fmt.Sprintf("%s (%d, %d)", t.Tok.TokenName, t.Extra, t.Extra2)
}

func indent(n int) string {
	return strings.Repeat("\t", n)
}

type GeneratorOutput struct {
	file *os.File
	data string
}

func NewGeneratorOutputFile(filename string) *GeneratorOutput {
	return &GeneratorOutput{openFile(filename), ""}
}

func NewGeneratorOutputString() *GeneratorOutput {
	return &GeneratorOutput{nil, ""}
}

func (g *GeneratorOutput) GetOutput() []byte {
	if g.file == nil {
		return []byte(g.data)
	}
	return []byte{}
}

func (g *GeneratorOutput) Close() {
	if g.file != nil {
		g.file.Close()
	}
}

func (g *GeneratorOutput) Printf(format string, a ...interface{}) {
	if g.file == nil {
		g.data += fmt.Sprintf(format, a...)
	} else {
		fmt.Fprintf(g.file, format, a...)
	}
}

func (g *GeneratorOutput) Println(text string) {
	if g.file == nil {
		g.data += text + "\n"
	} else {
		fmt.Fprintln(g.file, text)
	}
}

func (g *GeneratorOutput) Print(text string) {
	if g.file == nil {
		g.data += text
	} else {
		fmt.Fprint(g.file, text)
	}
}

func openFile(filename string) *os.File {
	if filename == "" || filename == "-" {
		return os.Stdout
	}

	// Open file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		os.Exit(1)
	}

	return file
}
