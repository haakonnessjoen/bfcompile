package main

import (
	"flag"
	"fmt"
	"os"

	"bcomp/bfutils"
	g "bcomp/generators"
	i "bcomp/interpreter"
	p "bcomp/parser"
)

var (
	optGenerator    string
	optInterpret    bool
	optOptimize     bool
	optDebug        bool
	optDebugSymbols bool
	optComments     bool
	optWordSize     int
	optMemorySize   int
	optOutput       string
)

const PACKAGE_NAME = "bfcompile"
const PACKAGE_VERSION = "1.0.0"

func main() {
	bfutils.Globals.Set("PACKAGE_NAME", PACKAGE_NAME)
	bfutils.Globals.Set("PACKAGE_VERSION", PACKAGE_VERSION)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// parse command line arguments
	flag.StringVar(&optGenerator, "g", "tokens", "Code generator to use: tokens, llvm, qbe, c, carm64, camd64, darwin-arm64, linux-amd64, js, rust or bf")
	flag.BoolVar(&optInterpret, "i", false, "Interpret the code instead of generating code. This will ignore the -g option.")
	flag.BoolVar(&optOptimize, "o", false, "Optimize the code")
	flag.BoolVar(&optComments, "c", false, "Add reference comments to the generated code")
	flag.BoolVar(&optDebug, "d", false, "Enable verbose output from optimizer")
	flag.BoolVar(&optDebugSymbols, "lg", false, "Enable LLVM debug symbols generation")
	flag.IntVar(&optWordSize, "w", 8, "Cell size (8, 16, 32 or 64)")
	flag.IntVar(&optMemorySize, "m", 30000, "Memory size available to brainfuck in the generated code")
	flag.StringVar(&optOutput, "out", "", "Set a filename to output to instead of outputting to STDOUT.")

	if optInterpret {
		optGenerator = "qbe"
	}

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

	if optWordSize != 8 && optWordSize != 16 && optWordSize != 32 && optWordSize != 64 {
		fmt.Fprintf(os.Stderr, "Error: Unknown cell size: %d\n\n", optWordSize)
		flag.Usage()
		os.Exit(1)
	}

	if optGenerator != "qbe" && optGenerator != "c" && optGenerator != "carm64" && optGenerator != "camd64" && optGenerator != "darwin-arm64" && optGenerator != "linux-amd64" && optGenerator != "js" && optGenerator != "rust" && optGenerator != "bf" && optGenerator != "tokens" && optGenerator != "llvm" {
		fmt.Fprintf(os.Stderr, "Error: Unknown generator %s\n\n", optGenerator)
		flag.Usage()
		os.Exit(1)
	}

	if optDebugSymbols && optGenerator != "llvm" {
		fmt.Fprintf(os.Stderr, "Error: -lg parameter is only relevant with the LLVM IR code generator")
		flag.Usage()
		os.Exit(1)
	}

	if optDebugSymbols {
		bfutils.Globals.Set("LLVM_DEBUG", "true")
	} else {
		bfutils.Globals.Set("LLVM_DEBUG", "false")
	}

	if optDebug {
		p.Debug = true
	}

	bfutils.Globals.Set("INPUT_FILENAME", flag.Args()[0])

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
		tokens = p.Optimize2(tokens, optGenerator, optWordSize)
		tokens = p.Optimize2(tokens, optGenerator, optWordSize)

		for {
			newtokens := p.Optimize(tokens)
			if len(newtokens) == len(tokens) {
				break
			}
			tokens = newtokens
		}
	}

	if optOptimize && initialCount > 0 {
		if optGenerator != "bf" {
			if optDebug {
				fmt.Fprintf(os.Stderr, "(bf output) Optimized from %d to %d instructions. Token reduction of %.f%%\n", initialCount, len(tokens), 100-((float64(len(tokens))/float64(initialCount))*100))
			}
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
			if optDebug {
				fmt.Fprintf(os.Stderr, "Optimized from %d to %d instructions. Reduction of %.f%%\n", initialCount, operations, 100-((float64(operations)/float64(initialCount))*100))
			}
		}
	}

	if optInterpret {
		i.InterpretTokens(tokens, optMemorySize, os.Stdin, bfutils.WrapStdout(os.Stdout), optWordSize)
	} else if optGenerator == "darwin-arm64" {
		g.PrintDarwinARM64(optOutput, tokens, optMemorySize, optWordSize)
	} else if optGenerator == "linux-amd64" {
		g.PrintLinuxAMD64(optOutput, tokens, optMemorySize, optWordSize)
	} else {
		output := g.NewGeneratorOutputFile(optOutput)
		defer output.Close()

		switch optGenerator {
		case "llvm":
			g.PrintIR(output, tokens, optComments, optMemorySize, optWordSize)
		case "qbe":
			g.PrintIL(output, tokens, optComments, optMemorySize, optWordSize)
		case "c":
			g.PrintC(output, tokens, optComments, optMemorySize, optWordSize)
		case "carm64":
			g.PrintCARM64(output, tokens, optComments, optMemorySize, optWordSize)
		case "camd64":
			g.PrintCAMD64(output, tokens, optComments, optMemorySize, optWordSize)
		case "js":
			g.PrintJS(output, tokens, optComments, optMemorySize, optWordSize)
		case "rust":
			g.PrintRust(output, tokens, optComments, optMemorySize, optWordSize)
		case "bf":
			g.PrintBF(output, tokens, optComments)
		case "tokens":
			g.PrintTokens(output, tokens, optComments)
		}
	}
}
