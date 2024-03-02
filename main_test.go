package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	g "bcomp/generators"
	p "bcomp/parser"

	"github.com/google/uuid"
)

func wantOutput(test string) string {
	content, err := os.ReadFile(fmt.Sprintf("testdata/%s_out.txt", test))
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
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

func (f TempFile) ReadFile() string {
	content, err := os.ReadFile(string(f))
	if err != nil {
		log.Fatal(err)
	}

	return string(content)
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

	if got != want {
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

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSVanilla(t *testing.T) {
	tokens := p.ParseFile("testdata/test03.bf")
	f := g.NewGeneratorOutputString()
	g.PrintJS(f, tokens, false, 30000)

	got := f.GetOutput()
	want := wantOutput("test03")

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSL1Optimized(t *testing.T) {
	tokens := p.ParseFile("testdata/test04.bf")
	tokens = p.Optimize(tokens)
	f := g.NewGeneratorOutputString()
	g.PrintJS(f, tokens, false, 30000)

	got := f.GetOutput()
	want := wantOutput("test04")

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSL2Optimized(t *testing.T) {
	tokens := p.ParseFile("testdata/test05.bf")
	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "js")
	f := g.NewGeneratorOutputString()
	g.PrintJS(f, tokens, false, 30000)

	got := f.GetOutput()
	want := wantOutput("test05")

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

// This test checks if the optimized loop outputs the correct code
// The code will go out of bounds if the first inner loop is not skipped
// correctly.
func TestBZCheckOfOptimizedLoops(t *testing.T) {
	tokens := p.ParseFile("testdata/test06.bf")
	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens, "js")

	tokenstrings := make([]string, 0, len(tokens))
	for _, t := range tokens {
		tokenstrings = append(tokenstrings, t.String())
	}
	gotb, err := json.MarshalIndent(tokenstrings, "", "\t")
	if err != nil {
		t.Errorf("json.Marshal: %v", err)
	}
	got := string(gotb)
	want := wantOutput("test06")

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
