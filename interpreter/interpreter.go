package interpreter

import (
	g "bcomp/generators"
	l "bcomp/lexer"
	"bufio"
	"fmt"
	"os"
)

type Jump struct {
	From int
	To   int
}

func InterpretTokens(tokens []g.ParseToken, memorySize int) {
	in, out := bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout)
	mem := make([]byte, memorySize)
	jumpLabels := make(map[int]Jump)

	// Compile jump-table
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		token := t.Tok.Tok
		jumplabel := t.Extra

		switch token {
		case l.JMPF:
			if (jumpLabels[jumplabel] == Jump{}) {
				jumpLabels[jumplabel] = Jump{From: i, To: 0}
			} else {
				jumpLabels[jumplabel] = Jump{From: i, To: jumpLabels[jumplabel].To}
			}
		case l.JMPB:
			if (jumpLabels[jumplabel] == Jump{}) {
				fmt.Fprintln(os.Stderr, "Warning: Unmatched jump label!")
				jumpLabels[jumplabel] = Jump{From: 0, To: i}
			} else {
				jumpLabels[jumplabel] = Jump{From: jumpLabels[jumplabel].From, To: i}
			}
		case l.LBL:
			if (jumpLabels[jumplabel] == Jump{}) {
				jumpLabels[jumplabel] = Jump{From: i, To: i}
			}
		}
	}

	// Run program
	p := 0
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		token := t.Tok.Tok
		value := t.Extra
		pointer := t.Extra2

		//fmt.Fprintf(os.Stderr, "%d Token: %v\n", i, t)

		switch token {
		case l.ADD:
			mem[p] += byte(value)
		case l.SUB:
			mem[p] -= byte(value)
		case l.INCP:
			p += value
		case l.DECP:
			p -= value
		case l.OUT:
			//fmt.Fprintf(os.Stderr, "Output: %c %d\n", mem[p], mem[p])
			for j := 0; j < value; j++ {
				out.WriteByte(mem[p])
			}
			out.Flush()
		case l.IN:
			for j := 0; j < value; j++ {
				c, err := in.ReadByte()
				if err != nil {
					mem[p] = 255
				} else {
					mem[p] = c
				}
			}
		case l.JMPF:
			if mem[p] == 0 {
				i = jumpLabels[value].To
				continue
			}
		case l.JMPB:
			if mem[p] != 0 {
				i = jumpLabels[value].From
				continue
			}
		case l.MUL:
			if value == -1 && pointer == 0 {
				mem[p] = 0
				continue
			}

			mem[p+pointer] += mem[p] * byte(value)
		case l.DIV:
			mem[p+pointer] /= byte(value)

		case l.BZ:
			if mem[p] == 0 {
				i = jumpLabels[value].To
				continue
			}
		case l.LBL:
			continue

		case l.MOV:
			mem[p+pointer] = byte(value)

		default:
			fmt.Fprintf(os.Stderr, "Warning: Unrecognized token: %v!\n", t.Tok.TokenName)
		}
	}

}
