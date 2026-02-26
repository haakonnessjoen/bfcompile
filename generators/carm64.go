package generators

import (
	l "bcomp/lexer"
	"fmt"
	"log"
)

type loopInfo struct {
	topIdx int // index of first instruction in JMPF block (the load)
	cbzIdx int // index of CBZ to patch
}

type arm64Gen struct {
	code      []uint32
	wordSize  int
	byteScale int // wordSize / 8
	sf        int // 0 for 8/16/32-bit cell ops, 1 for 64-bit
	loopStack []loopInfo
	ifPatch   map[int]int // BZ label -> CBZ instruction index
}

func (g *arm64Gen) emit(instr uint32) {
	g.code = append(g.code, instr)
}

func (g *arm64Gen) pos() int {
	return len(g.code)
}

// emitLoadCell loads the cell at [x19] into w9/x9
func (g *arm64Gen) emitLoadCell() {
	switch g.wordSize {
	case 8:
		g.emit(encLdrbUoff(X9, X19, 0))
	case 16:
		g.emit(encLdrhUoff(X9, X19, 0))
	case 32:
		g.emit(encLdrwUoff(X9, X19, 0))
	case 64:
		g.emit(encLdrxUoff(X9, X19, 0))
	}
}

// emitStoreCell stores w9/x9 to [x19]
func (g *arm64Gen) emitStoreCell() {
	switch g.wordSize {
	case 8:
		g.emit(encStrbUoff(X9, X19, 0))
	case 16:
		g.emit(encStrhUoff(X9, X19, 0))
	case 32:
		g.emit(encStrwUoff(X9, X19, 0))
	case 64:
		g.emit(encStrxUoff(X9, X19, 0))
	}
}

// emitLoadTo loads the cell at [x19 + offset*byteScale] into register rt
func (g *arm64Gen) emitLoadTo(rt int, offset int) {
	byteOff := offset * g.byteScale
	if byteOff >= 0 && byteOff%g.byteScale == 0 {
		scaled := uint32(byteOff / g.byteScale)
		if scaled <= 0xFFF {
			switch g.wordSize {
			case 8:
				g.emit(encLdrbUoff(rt, X19, scaled))
			case 16:
				g.emit(encLdrhUoff(rt, X19, scaled))
			case 32:
				g.emit(encLdrwUoff(rt, X19, scaled))
			case 64:
				g.emit(encLdrxUoff(rt, X19, scaled))
			}
			return
		}
	}
	if byteOff >= -256 && byteOff <= 255 {
		switch g.wordSize {
		case 8:
			g.emit(encLdurb(rt, X19, byteOff))
		case 16:
			g.emit(encLdurh(rt, X19, byteOff))
		case 32:
			g.emit(encLdurw(rt, X19, byteOff))
		case 64:
			g.emit(encLdurx(rt, X19, byteOff))
		}
		return
	}
	// General case: compute address in x11
	if byteOff > 0 {
		emitAddImm(&g.code, 1, X11, X19, byteOff, X10)
	} else {
		emitSubImm(&g.code, 1, X11, X19, -byteOff, X10)
	}
	switch g.wordSize {
	case 8:
		g.emit(encLdrbUoff(rt, X11, 0))
	case 16:
		g.emit(encLdrhUoff(rt, X11, 0))
	case 32:
		g.emit(encLdrwUoff(rt, X11, 0))
	case 64:
		g.emit(encLdrxUoff(rt, X11, 0))
	}
}

// emitStoreTo stores register rt to [x19 + offset*byteScale]
func (g *arm64Gen) emitStoreTo(rt int, offset int) {
	byteOff := offset * g.byteScale
	if byteOff >= 0 && byteOff%g.byteScale == 0 {
		scaled := uint32(byteOff / g.byteScale)
		if scaled <= 0xFFF {
			switch g.wordSize {
			case 8:
				g.emit(encStrbUoff(rt, X19, scaled))
			case 16:
				g.emit(encStrhUoff(rt, X19, scaled))
			case 32:
				g.emit(encStrwUoff(rt, X19, scaled))
			case 64:
				g.emit(encStrxUoff(rt, X19, scaled))
			}
			return
		}
	}
	if byteOff >= -256 && byteOff <= 255 {
		switch g.wordSize {
		case 8:
			g.emit(encSturb(rt, X19, byteOff))
		case 16:
			g.emit(encSturh(rt, X19, byteOff))
		case 32:
			g.emit(encSturw(rt, X19, byteOff))
		case 64:
			g.emit(encSturx(rt, X19, byteOff))
		}
		return
	}
	// General case: compute address in x11 (same as load)
	if byteOff > 0 {
		emitAddImm(&g.code, 1, X11, X19, byteOff, X10)
	} else {
		emitSubImm(&g.code, 1, X11, X19, -byteOff, X10)
	}
	switch g.wordSize {
	case 8:
		g.emit(encStrbUoff(rt, X11, 0))
	case 16:
		g.emit(encStrhUoff(rt, X11, 0))
	case 32:
		g.emit(encStrwUoff(rt, X11, 0))
	case 64:
		g.emit(encStrxUoff(rt, X11, 0))
	}
}

// emitMask masks register rd to the current word size (8 or 16 bit)
func (g *arm64Gen) emitMask(rd int) {
	switch g.wordSize {
	case 8:
		g.emit(encAndImm8(rd, rd))
	case 16:
		g.emit(encAndImm16(rd, rd))
	}
}

func (g *arm64Gen) emitPrologue() {
	// STP x29, x30, [sp, #-48]!
	g.emit(encStpPre64(X29, X30, SP, -6)) // -6 * 8 = -48
	// MOV x29, sp
	g.emit(encAddImm(1, X29, SP, 0, 0))
	// STP x19, x20, [sp, #16]
	g.emit(encStpOff64(X19, X20, SP, 2)) // 2 * 8 = 16
	// STP x21, xzr, [sp, #32]
	g.emit(encStpOff64(X21, XZR, SP, 4)) // 4 * 8 = 32
	// MOV x19, x0 (mem pointer)
	g.emit(encMovReg(1, X19, X0))
	// MOV x20, x1 (putchar)
	g.emit(encMovReg(1, X20, X1))
	// MOV x21, x2 (getchar)
	g.emit(encMovReg(1, X21, X2))
}

func (g *arm64Gen) emitEpilogue() {
	// LDP x19, x20, [sp, #16]
	g.emit(encLdpOff64(X19, X20, SP, 2))
	// LDP x21, xzr, [sp, #32]
	g.emit(encLdpOff64(X21, XZR, SP, 4))
	// LDP x29, x30, [sp], #48
	g.emit(encLdpPost64(X29, X30, SP, 6))
	// RET
	g.emit(encRet())
}

// PrintCARM64 generates a C file containing ARM64 machine code as a uint32_t array
func PrintCARM64(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
	gen := &arm64Gen{
		code:      make([]uint32, 0, 1024),
		wordSize:  wordSize,
		byteScale: wordSize / 8,
		loopStack: make([]loopInfo, 0, 32),
		ifPatch:   make(map[int]int),
	}

	if wordSize == 64 {
		gen.sf = 1
	}

	gen.emitPrologue()

	for _, t := range tokens {
		switch t.Tok.Tok {
		case l.ADD:
			gen.emitLoadCell()
			emitAddImm(&gen.code, gen.sf, X9, X9, t.Extra, X11)
			gen.emitMask(X9)
			gen.emitStoreCell()

		case l.SUB:
			gen.emitLoadCell()
			emitSubImm(&gen.code, gen.sf, X9, X9, t.Extra, X11)
			gen.emitMask(X9)
			gen.emitStoreCell()

		case l.INCP:
			byteOff := t.Extra * gen.byteScale
			emitAddImm(&gen.code, 1, X19, X19, byteOff, X11)

		case l.DECP:
			byteOff := t.Extra * gen.byteScale
			emitSubImm(&gen.code, 1, X19, X19, byteOff, X11)

		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				gen.emitLoadCell()
				gen.emit(encMovReg(0, X0, X9))
				gen.emit(encBlr(X20))
			}

		case l.IN:
			for i := 0; i < t.Extra; i++ {
				gen.emit(encBlr(X21))
			}
			gen.emit(encMovReg(gen.sf, X9, X0))
			gen.emitMask(X9)
			gen.emitStoreCell()

		case l.JMPF:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.loopStack = append(gen.loopStack, loopInfo{topIdx, cbzIdx})

		case l.JMPB:
			info := gen.loopStack[len(gen.loopStack)-1]
			gen.loopStack = gen.loopStack[:len(gen.loopStack)-1]
			// B back to the load at topIdx
			gen.emit(encB(info.topIdx - gen.pos()))
			// Patch CBZ to jump here (past the B)
			gen.code[info.cbzIdx] = encCbz(gen.sf, X9, gen.pos()-info.cbzIdx)

		case l.BZ:
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.ifPatch[t.Extra] = cbzIdx

		case l.LBL:
			cbzIdx := gen.ifPatch[t.Extra]
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		case l.MUL:
			multiplier := t.Extra
			offset := t.Extra2

			// w9 = *p
			gen.emitLoadCell()

			if multiplier == 1 {
				// p[offset] += *p
				gen.emitLoadTo(X10, offset)
				gen.emit(encAddReg(gen.sf, X10, X10, X9))
			} else if multiplier == -1 {
				// p[offset] -= *p
				gen.emitLoadTo(X10, offset)
				gen.emit(encSubReg(gen.sf, X10, X10, X9))
			} else {
				// p[offset] += *p * multiplier
				abs := multiplier
				if abs < 0 {
					abs = -abs
				}
				emitMovImm(&gen.code, gen.sf, X11, uint64(abs))
				// x11 = x9 * x11 (MUL = MADD with Ra=XZR)
				gen.emit(encMadd(gen.sf, X11, X9, X11, XZR))
				gen.emitLoadTo(X10, offset)
				if multiplier > 0 {
					gen.emit(encAddReg(gen.sf, X10, X10, X11))
				} else {
					gen.emit(encSubReg(gen.sf, X10, X10, X11))
				}
			}
			gen.emitMask(X10)
			gen.emitStoreTo(X10, offset)

		case l.DIV:
			divisor := t.Extra
			offset := t.Extra2

			gen.emitLoadTo(X9, offset)
			emitMovImm(&gen.code, gen.sf, X11, uint64(divisor))
			gen.emit(encUdiv(gen.sf, X9, X9, X11))
			gen.emitMask(X9)
			gen.emitStoreTo(X9, offset)

		case l.MOV:
			value := t.Extra
			offset := t.Extra2

			if value == 0 {
				gen.emitStoreTo(XZR, offset)
			} else {
				emitMovImm(&gen.code, gen.sf, X9, uint64(value))
				gen.emitMask(X9)
				gen.emitStoreTo(X9, offset)
			}

		case l.SCANR:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.emit(encAddImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		case l.SCANL:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.emit(encSubImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		case l.PRNT:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.emit(encMovReg(0, X0, X9))
			gen.emit(encBlr(X20))
			gen.emit(encAddImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}

	gen.emitEpilogue()

	// Output the C wrapper
	wordType := ""
	switch wordSize {
	case 8:
		wordType = "uint8_t"
	case 16:
		wordType = "uint16_t"
	case 32:
		wordType = "uint32_t"
	case 64:
		wordType = "uint64_t"
	}

	f.Println("/* Generated by bfcompile - ARM64 native code generator */")
	f.Println("#include <stdio.h>")
	f.Println("#include <stdint.h>")
	f.Println("#include <string.h>")
	f.Println("#include <sys/mman.h>")
	f.Println("#include <libkern/OSCacheControl.h>")
	f.Println("#include <pthread.h>")
	f.Println("")
	f.Printf("static %s mem[%d];\n\n", wordType, memorySize)
	f.Println("static const uint32_t bf_code[] = {")
	for i, instr := range gen.code {
		if i%8 == 0 {
			f.Print("    ")
		}
		if i == len(gen.code)-1 {
			f.Printf("0x%08X", instr)
		} else if i%8 == 7 {
			f.Printf("0x%08X,\n", instr)
		} else {
			f.Printf("0x%08X, ", instr)
		}
	}
	f.Println("")
	f.Println("};")
	f.Println("")
	f.Println("int main(void) {")
	f.Println("    size_t code_size = sizeof(bf_code);")
	f.Println("    size_t page = 16384;")
	f.Println("    size_t alloc = (code_size + page - 1) & ~(page - 1);")
	f.Println("    void *mem_exec = mmap(NULL, alloc, PROT_READ | PROT_WRITE | PROT_EXEC,")
	f.Println("                          MAP_PRIVATE | MAP_ANON | MAP_JIT, -1, 0);")
	f.Println("    if (mem_exec == MAP_FAILED) { perror(\"mmap\"); return 1; }")
	f.Println("    pthread_jit_write_protect_np(0);")
	f.Println("    memcpy(mem_exec, bf_code, code_size);")
	f.Println("    pthread_jit_write_protect_np(1);")
	f.Println("    sys_icache_invalidate(mem_exec, code_size);")
	f.Printf("    ((void (*)(void *, int (*)(int), int (*)(void)))mem_exec)(mem, putchar, getchar);\n")
	f.Println("    munmap(mem_exec, alloc);")
	f.Println("    return 0;")
	f.Println("}")
	if includeComments {
		f.Printf("/* %d ARM64 instructions generated */\n", len(gen.code))
	}
}

// formatHex is a helper for hex formatting
func formatHex(v uint32) string {
	return fmt.Sprintf("0x%08X", v)
}
