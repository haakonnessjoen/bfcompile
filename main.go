package main

import (
	"flag"
	"fmt"
	"os"

	g "bcomp/generators"
)

var (
	optGenerator string
	optOptimize  bool
	optComments  bool
)

func main() {
	// parse command line arguments
	flag.StringVar(&optGenerator, "g", "qbe", "Generator to use, qbe, c, js or bf")
	flag.BoolVar(&optOptimize, "o", false, "Optimize the code")
	flag.BoolVar(&optComments, "c", false, "Add reference comments to the generated code")

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

	tokens := parseFile(flag.Args()[0])
	initialCount := len(tokens)

	if optOptimize {
		for {
			newtokens := optimize(tokens)
			if len(newtokens) == len(tokens) {
				// No more optimization to be done
				break
			}
			// We managed to remove some instructions, try again
			tokens = newtokens
		}
	}

	if optOptimize && optGenerator != "bf" {
		tokens = optimize2(tokens)
	}

	if optOptimize && initialCount > 0 {
		if optGenerator != "bf" {
			fmt.Fprintf(os.Stderr, "Optimized from %d to %d instructions. Token reduction of %.f%%\n", initialCount, len(tokens), 100-((float64(len(tokens))/float64(initialCount))*100))
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
			fmt.Fprintf(os.Stderr, "Optimized from %d to %d instructions. Reduction of %.f%%\n", initialCount, operations, 100-((float64(operations)/float64(initialCount))*100))
		}
	}

	switch optGenerator {
	case "qbe":
		g.PrintIL(tokens, optComments)
	case "c":
		g.PrintC(tokens, optComments)
	case "js":
		g.PrintJS(tokens, optComments)
	case "bf":
		g.PrintBF(tokens, optComments)
	}
}
