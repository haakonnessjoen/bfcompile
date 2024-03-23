package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"bcomp/bfutils"
	g "bcomp/generators"
	i "bcomp/interpreter"
	p "bcomp/parser"

	"github.com/google/uuid"
)

func init() {
	g.PrintWarnings = false
	i.PrintWarnings = false
}

func wantOutput(test string) []byte {
	content, err := os.ReadFile(fmt.Sprintf("testdata/%s_out.txt", test))
	if err != nil {
		log.Fatal(err)
	}
	return content
}

type TempFile string

func tempFile() TempFile {
	UUID, err := uuid.NewV7()
	if err != nil {
		log.Fatal(err)
	}
	path := os.TempDir()
	file := fmt.Sprintf("%s%s", path, UUID.String())
	return TempFile(file)
}

func getInterpretedOutput(tokens []g.ParseToken, input []byte) []byte {
	in := bytes.NewReader(input)
	out := bytes.NewBuffer([]byte{})
	i.InterpretTokens(tokens, 30000, in, bfutils.WrapBuffer(out), 8)

	return out.Bytes()
}

func (f TempFile) ReadFile() []byte {
	content, err := os.ReadFile(string(f))
	if err != nil {
		log.Fatal(err)
	}

	return content
}

func (f TempFile) Remove() {
	os.Remove(string(f))
}

func TestBFtoBF(t *testing.T) {
	oldArgs := os.Args

	defer func() {
		os.Args = oldArgs
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()

	filename := tempFile()
	defer filename.Remove()
	os.Args = []string{"cmd", "-g", "bf", "-out", string(filename), "testdata/test01.bf"}
	main()
	got := filename.ReadFile()
	want := wantOutput("test01")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestBFOptimizetoBF(t *testing.T) {
	oldArgs := os.Args

	defer func() {
		os.Args = oldArgs
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()

	filename := tempFile()
	defer filename.Remove()
	os.Args = []string{"cmd", "-o", "-g", "bf", "-out", string(filename), "testdata/test02.bf"}
	main()
	got := filename.ReadFile()
	want := wantOutput("test02")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeNumasciiart(t *testing.T) {
	tokens := p.ParseFile("brainfuck/numasciiart.bf")

	got := getInterpretedOutput(tokens, []byte("(0123456789-abcdef/. . .)\n"))
	want := wantOutput("numasciiart")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeNumasciiartOptimize1(t *testing.T) {
	tokens := p.ParseFile("brainfuck/numasciiart.bf")

	tokens = p.Optimize(tokens)

	got := getInterpretedOutput(tokens, []byte("(0123456789-abcdef/. . .)\n"))
	want := wantOutput("numasciiart")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeNumasciiartOptimize2(t *testing.T) {
	tokens := p.ParseFile("brainfuck/numasciiart.bf")

	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "")

	got := getInterpretedOutput(tokens, []byte("(0123456789-abcdef/. . .)\n"))
	want := wantOutput("numasciiart")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeTictactoe(t *testing.T) {
	tokens := p.ParseFile("brainfuck/tictactoe.bf")

	got := getInterpretedOutput(tokens, []byte("5\n8\n3\n4\n"))
	want := wantOutput("tictactoe")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeTictactoeOptimize1(t *testing.T) {
	tokens := p.ParseFile("brainfuck/tictactoe.bf")

	tokens = p.Optimize(tokens)

	got := getInterpretedOutput(tokens, []byte("5\n8\n3\n4\n"))
	want := wantOutput("tictactoe")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestComplicatedCodeTictactoeOptimize2(t *testing.T) {
	tokens := p.ParseFile("brainfuck/tictactoe.bf")

	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "")
	tokens = p.Optimize2(tokens, "")

	got := getInterpretedOutput(tokens, []byte("5\n8\n3\n4\n"))
	want := wantOutput("tictactoe")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSL1Optimized(t *testing.T) {
	tokens := p.ParseFile("testdata/test04.bf")

	for {
		newtokens := p.Optimize(tokens)
		if len(newtokens) == len(tokens) {
			break
		}
		tokens = newtokens
	}

	f := g.NewGeneratorOutputString()
	g.PrintJS(f, tokens, false, 30000, 8)

	got := f.GetOutput()
	want := wantOutput("test04")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSL2Optimized(t *testing.T) {
	tokens := p.ParseFile("testdata/test05.bf")

	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "js")
	f := g.NewGeneratorOutputString()
	g.PrintJS(f, tokens, false, 30000, 8)

	got := f.GetOutput()
	want := wantOutput("test05")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

// This test checks if the optimized loop outputs the correct code
// The code will go out of bounds if the first inner loop is not skipped
// correctly.
func TestBZCheckComplicatedCode1(t *testing.T) {
	tokens := p.ParseFile("testdata/test06.bf")
	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "js")

	tokenstrings := make([]string, 0, len(tokens))
	for _, t := range tokens {
		tokenstrings = append(tokenstrings, t.String())
	}
	got, err := json.MarshalIndent(tokenstrings, "", "\t")
	if err != nil {
		t.Errorf("json.Marshal: %v", err)
	}
	want := wantOutput("test06")

	if !bytes.Equal(got, want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
