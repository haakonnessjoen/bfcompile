package generators

import (
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
	"strings"
)

// PrintJS prints the tokens as node.js code
func PrintJS(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int) {
	hasInput := false
	for _, t := range tokens {
		if t.Tok.Tok == l.IN {
			hasInput = true
			break
		}
	}

	f.Println(`const process = require("process");`)
	if hasInput {
		f.Println(`const inputcb = [];
const inputbuf = [];

async function input() {
	if (inputbuf.length > 0) {
		return inputbuf.shift();
	}

	return new Promise((resolve) => {
		inputcb.push((v) => {
			resolve(v);
		});
	});
}

process.stdin.on("data", (data) => {
	for (let i = 0; i < data.length; i++) {
		if (inputcb.length > 0) {
			const cb = inputcb.shift();
			cb(data[i]);
		} else {
			inputbuf.push(data[i]);
		}
	}
});`)
	}

	f.Printf(`async function output(v) {
	let wrote = process.stdout.write(String.fromCharCode(v));
	if (!wrote) {
		await new Promise((resolve) => {
			process.stdout.once("drain", resolve);
		});
		output(v);
	}
}

async function main() {
	const mem = new Uint8Array(%d);
	let p = 0;`, memorySize)

	indentLevel := 1
	for _, t := range tokens {
		if includeComments {
			f.Printf("%s// Line %d, Pos %d: %v\n", indent(indentLevel), t.Pos.Line, t.Pos.Column, t.Tok)
		}

		switch t.Tok.Tok {
		case l.ADD:
			if t.Extra == 1 {
				f.Printf("%smem[p]++;\n", indent(indentLevel))
			} else {
				f.Printf("%smem[p] += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.SUB:
			if t.Extra == 1 {
				f.Printf("%smem[p]--;\n", indent(indentLevel))
			} else {
				f.Printf("%smem[p] -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.INCP:
			if t.Extra == 1 {
				f.Printf("%sp++;\n", indent(indentLevel))
			} else {
				f.Printf("%sp += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.DECP:
			if t.Extra == 1 {
				f.Printf("%sp--;\n", indent(indentLevel))
			} else {
				f.Printf("%sp -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.OUT:
			if t.Extra == 1 {
				f.Printf("%sawait output(mem[p]);\n", indent(indentLevel))
			} else {
				f.Printf("%sfor (let i = 0; i < %d; i++) {\n%s	await output(mem[p]);\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
		case l.IN:
			if t.Extra == 1 {
				f.Printf("%smem[p] = await input();\n", indent(indentLevel))
			} else {
				f.Printf("%sfor (let i = 0; i < %d; i++) {\n%s	mem[p] = await input();\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
		case l.JMPF:
			f.Printf("%swhile (mem[p]) {\n", indent(indentLevel))
			indentLevel++
		case l.JMPB:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		case l.MUL:
			if t.Extra == -1 && t.Extra2 == 0 {
				f.Printf("%smem[p] = 0;\n", indent(indentLevel))
				continue
			}

			prefix := ""
			if t.Extra2 >= 0 {
				prefix = "+"
			}
			output := ""
			if t.Extra == 1 {
				output = fmt.Sprintf("%smem[p%s%d] += mem[p];\n", indent(indentLevel), prefix, t.Extra2)
			} else if t.Extra == -1 {
				output = fmt.Sprintf("%smem[p%s%d] -= mem[p];\n", indent(indentLevel), prefix, t.Extra2)
			} else {
				output = fmt.Sprintf("%smem[p%s%d] += mem[p] * %d;\n", indent(indentLevel), prefix, t.Extra2, t.Extra)
			}
			f.Print(strings.ReplaceAll(output, "mem[p+0]", "mem[p]"))
		case l.DIV:
			prefix := ""
			if t.Extra2 >= 0 {
				prefix = "+"
			}
			output := ""
			output = fmt.Sprintf("%smem[p%s%d] /= %d;\n", indent(indentLevel), prefix, t.Extra2, t.Extra)
			f.Print(strings.ReplaceAll(output, "mem[p+0]", "mem[p]"))
		case l.BZ:
			f.Printf("%sif (mem[p]) {\n", indent(indentLevel))
			indentLevel++
		case l.LBL:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		case l.MOV:
			if t.Extra2 == 0 {
				f.Printf("%smem[p] = %d;\n", indent(indentLevel), t.Extra)
			} else {
				f.Printf("%smem[p+%d] = %d;\n", indent(indentLevel), t.Extra2, t.Extra)
			}
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	if indentLevel > 1 {
		fmt.Fprintf(os.Stderr, "Warning: Unbalanced brackets in code\n")
		for indentLevel > 1 {
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		}
	}
	f.Println("	process.stdin.unref();")
	f.Println("}")
	f.Println("main()")
}
