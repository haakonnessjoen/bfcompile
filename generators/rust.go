package generators

import (
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
	"strings"
)

// PrintRust prints the tokens as Rust code
func PrintRust(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
	wordType := ""
	switch wordSize {
	case 8:
		wordType = "u8"
	case 16:
		wordType = "u16"
	case 32:
		wordType = "u32"
	case 64:
		wordType = "u64"
	default:
		log.Fatalf("Error: Unknown word size %d\n", wordSize)
	}

	hasInput := false
	for _, t := range tokens {
		if t.Tok.Tok == l.IN {
			hasInput = true
			break
		}
	}

	f.Println("use std::io::{self, Write};")
	if hasInput {
		f.Println("use std::io::Read;")
	}
	f.Println("")
	f.Println("fn main() {")
	f.Printf("\tlet mut mem = [0%s; %d];\n", wordType, memorySize)
	f.Println("\tlet mut p: usize = 0;")
	f.Println("")

	indentLevel := 1
	for _, t := range tokens {
		if includeComments {
			f.Printf("%s// Line %d, Pos %d: %v\n", indent(indentLevel), t.Pos.Line, t.Pos.Column, t.Tok)
		}

		switch t.Tok.Tok {
		case l.ADD:
			if t.Extra == 1 {
				f.Printf("%smem[p] = mem[p].wrapping_add(1);\n", indent(indentLevel))
			} else {
				f.Printf("%smem[p] = mem[p].wrapping_add(%d);\n", indent(indentLevel), t.Extra)
			}
		case l.SUB:
			if t.Extra == 1 {
				f.Printf("%smem[p] = mem[p].wrapping_sub(1);\n", indent(indentLevel))
			} else {
				f.Printf("%smem[p] = mem[p].wrapping_sub(%d);\n", indent(indentLevel), t.Extra)
			}
		case l.INCP:
			if t.Extra == 1 {
				f.Printf("%sp += 1;\n", indent(indentLevel))
			} else {
				f.Printf("%sp += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.DECP:
			if t.Extra == 1 {
				f.Printf("%sp -= 1;\n", indent(indentLevel))
			} else {
				f.Printf("%sp -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.OUT:
			if t.Extra == 1 {
				f.Printf("%sio::stdout().write_all(&[mem[p] as u8]).unwrap();\n", indent(indentLevel))
			} else {
				f.Printf("%sfor _ in 0..%d {\n", indent(indentLevel), t.Extra)
				f.Printf("%s\tio::stdout().write_all(&[mem[p] as u8]).unwrap();\n", indent(indentLevel))
				f.Printf("%s}\n", indent(indentLevel))
			}
		case l.IN:
			if t.Extra == 1 {
				f.Printf("%s{\n", indent(indentLevel))
				f.Printf("%s\tlet mut buf = [0u8; 1];\n", indent(indentLevel))
				f.Printf("%s\tio::stdin().read(&mut buf).unwrap();\n", indent(indentLevel))
				f.Printf("%s\tmem[p] = buf[0] as %s;\n", indent(indentLevel), wordType)
				f.Printf("%s}\n", indent(indentLevel))
			} else {
				f.Printf("%sfor _ in 0..%d {\n", indent(indentLevel), t.Extra)
				f.Printf("%s\tlet mut buf = [0u8; 1];\n", indent(indentLevel))
				f.Printf("%s\tio::stdin().read(&mut buf).unwrap();\n", indent(indentLevel))
				f.Printf("%s\tmem[p] = buf[0] as %s;\n", indent(indentLevel), wordType)
				f.Printf("%s}\n", indent(indentLevel))
			}
		case l.JMPF:
			f.Printf("%swhile mem[p] != 0 {\n", indent(indentLevel))
			indentLevel++
		case l.JMPB:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		case l.MUL:
			output := ""
			if t.Extra == 1 {
				output = fmt.Sprintf("%smem[p + %d] = mem[p + %d].wrapping_add(mem[p]);\n", indent(indentLevel), t.Extra2, t.Extra2)
			} else if t.Extra == -1 {
				output = fmt.Sprintf("%smem[p + %d] = mem[p + %d].wrapping_sub(mem[p]);\n", indent(indentLevel), t.Extra2, t.Extra2)
			} else {
				output = fmt.Sprintf("%smem[p + %d] = mem[p + %d].wrapping_add(mem[p].wrapping_mul(%d));\n", indent(indentLevel), t.Extra2, t.Extra2, t.Extra)
			}
			output = strings.ReplaceAll(output, "mem[p + 0]", "mem[p]")
			// Handle negative offsets
			output = strings.ReplaceAll(output, "p + -", "p - ")
			f.Print(output)
		case l.DIV:
			output := fmt.Sprintf("%smem[p + %d] /= %d;\n", indent(indentLevel), t.Extra2, t.Extra)
			output = strings.ReplaceAll(output, "mem[p + 0]", "mem[p]")
			output = strings.ReplaceAll(output, "p + -", "p - ")
			f.Print(output)
		case l.BZ:
			f.Printf("%sif mem[p] != 0 {\n", indent(indentLevel))
			indentLevel++
		case l.LBL:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		case l.SCANR:
			f.Printf("%sp += mem[p..].iter().position(|&x| x == 0).unwrap();\n", indent(indentLevel))
		case l.SCANL:
			f.Printf("%sp -= mem[..=p].iter().rev().position(|&x| x == 0).unwrap();\n", indent(indentLevel))
		case l.MOV:
			if t.Extra2 == 0 {
				f.Printf("%smem[p] = %d;\n", indent(indentLevel), t.Extra)
			} else {
				output := fmt.Sprintf("%smem[p + %d] = %d;\n", indent(indentLevel), t.Extra2, t.Extra)
				output = strings.ReplaceAll(output, "p + -", "p - ")
				f.Print(output)
			}
		case l.PRNT:
			f.Printf("%s{\n", indent(indentLevel))
			f.Printf("%s\tlet start = p;\n", indent(indentLevel))
			f.Printf("%s\twhile mem[p] != 0 {\n", indent(indentLevel))
			f.Printf("%s\t\tp += 1;\n", indent(indentLevel))
			f.Printf("%s\t}\n", indent(indentLevel))
			f.Printf("%s\tio::stdout().write_all(&mem[start..p].iter().map(|&x| x as u8).collect::<Vec<u8>>()).unwrap();\n", indent(indentLevel))
			f.Printf("%s}\n", indent(indentLevel))
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	if indentLevel > 1 {
		if PrintWarnings {
			fmt.Fprintf(os.Stderr, "Warning: Unbalanced brackets in code\n")
		}
		for indentLevel > 1 {
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		}
	}
	f.Println("\tio::stdout().flush().unwrap();")
	f.Println("}")
}
