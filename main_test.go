package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

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
	oldArgs := os.Args

	defer func() {
		os.Args = oldArgs
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()

	filename := tempFile()
	defer filename.Remove()
	os.Args = []string{"cmd", "-g", "js", "-out", string(filename), "testdata/test03.bf"}
	main()
	got := filename.ReadFile()
	want := wantOutput("test03")

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSOptimized(t *testing.T) {
	oldArgs := os.Args

	defer func() {
		os.Args = oldArgs
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()

	filename := tempFile()
	defer filename.Remove()
	os.Args = []string{"cmd", "-o", "-g", "js", "-out", string(filename), "testdata/test04.bf"}
	main()
	got := filename.ReadFile()
	want := wantOutput("test04")

	os.WriteFile("testdata/test04_out.txt", []byte(got), 0666)

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestJSINTOptimized(t *testing.T) {
	tokens := p.ParseFile("testdata/test05.bf")
	tokens = p.Optimize(tokens)
	tokens = p.Optimize2(tokens)

	got, _ := json.Marshal(tokens)
	want := wantOutput("test05")

	if string(got) != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
