package main

import (
	"flag"
	"fmt"
	"os"

	g "bcomp/generators"
	p "bcomp/parser"
)

var (
	optGenerator  string
	optOptimize   bool
	optDebug      bool
	optComments   bool
	optMemorySize int
	optOutput     string
)

func main() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// parse command line arguments
	flag.StringVar(&optGenerator, "g", "qbe", "Generator to use, qbe, c, js or bf")
	flag.BoolVar(&optOptimize, "o", false, "Optimize the code")
	flag.BoolVar(&optComments, "c", false, "Add reference comments to the generated code")
	flag.BoolVar(&optDebug, "d", false, "Enable debug output from optimizer")
	flag.IntVar(&optMemorySize, "m", 30000, "Memory size available to brainfuck in the generated code")
	flag.StringVar(&optOutput, "out", "", "Set a filename to output to instead of outputting to STDOUT.")

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

	if optGenerator != "qbe" && optGenerator != "c" && optGenerator != "js" && optGenerator != "bf" {
		fmt.Fprintf(os.Stderr, "Error: Unknown generator %s\n\n", optGenerator)
		flag.Usage()
		os.Exit(1)
	}

	if optDebug {
		p.Debug = true
	}

	tokens := p.ParseFile(flag.Args()[0])
	initialCount := len(tokens)

	if optOptimize {
		for {
			newtokens := p.Optimize(tokens)
			if len(newtokens) == len(tokens) {
				// No more optimization to be done
				break
			}
			// We managed to remove some instructions, try again
			tokens = newtokens
		}
	}

	if optOptimize && optGenerator != "bf" {
		tokens = p.Optimize2(tokens, optGenerator)
	}

	if optOptimize && initialCount > 0 {
		if optGenerator != "bf" {
			//fmt.Fprintf(os.Stderr, "Optimized from %d to %d instructions. Token reduction of %.f%%\n", initialCount, len(tokens), 100-((float64(len(tokens))/float64(initialCount))*100))
		} else {
			operations := 0
			for _, t := range tokens {
				if t.Tok.TokenName != "JMPF" && t.Tok.TokenName != "JMPB" {
					operations += t.Extra
				} else {
					operations++
				}
			}

			// To not confuse the user, we count all the individual instructions as brainfuck will not be able to output less instructions
			//fmt.Fprintf(os.Stderr, "Optimized from %d to %d instructions. Reduction of %.f%%\n", initialCount, operations, 100-((float64(operations)/float64(initialCount))*100))
		}
	}

	output := g.NewGeneratorOutputFile(optOutput)
	defer output.Close()

	switch optGenerator {
	case "qbe":
		g.PrintIL(output, tokens, optComments, optMemorySize)
	case "c":
		g.PrintC(output, tokens, optComments, optMemorySize)
	case "js":
		g.PrintJS(output, tokens, optComments, optMemorySize)
	case "bf":
		g.PrintBF(output, tokens, optComments)
	}
}
