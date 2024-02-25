package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	optGenerator string
	optOptimize  bool
)

func main() {
	// parse command line arguments
	flag.StringVar(&optGenerator, "g", "qbe", "Generator to use, qbe or c")
	flag.BoolVar(&optOptimize, "o", false, "Optimize the code")

	// Customize usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <brainfuck file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Missing filename of brainfuck file\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if optGenerator != "qbe" && optGenerator != "c" {
		fmt.Fprintf(os.Stderr, "Error: Unknown generator %s\n\n", optGenerator)
		flag.Usage()
		os.Exit(1)
	}

	tokens := parseFile(flag.Args()[0])

	if optOptimize {
		tokens = optimize(tokens)
	}

	switch optGenerator {
	case "qbe":
		PrintIL(tokens)
	case "c":
		PrintC(tokens)
	}
}
