package generators

// x86-64 register encoding numbers
const (
	RAX = 0
	RCX = 1
	RDX = 2
	RBX = 3
	RSP = 4
	RBP = 5
	RSI = 6
	RDI = 7
	R8  = 8
	R12 = 12
	R13 = 13
)

type amd64LoopInfo struct {
	topOff int // byte offset of loop top (the load)
	jeOff  int // byte offset of JE rel32 to patch
}

type amd64Gen struct {
	code      []byte
	wordSize  int
	byteScale int
	loopStack []amd64LoopInfo
	ifPatch   map[int]int // BZ label -> JE byte offset
	useFnPtr  bool        // true for camd64 (use r12/r13 for I/O)
}

func (g *amd64Gen) emit(bytes ...byte) {
	g.code = append(g.code, bytes...)
}

func (g *amd64Gen) emit32le(v uint32) {
	g.emit(byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func (g *amd64Gen) pos() int {
	return len(g.code)
}

func (g *amd64Gen) patch32(off int, v uint32) {
	g.code[off] = byte(v)
	g.code[off+1] = byte(v >> 8)
	g.code[off+2] = byte(v >> 16)
	g.code[off+3] = byte(v >> 24)
}

// rex returns a REX prefix byte. w=1 for 64-bit operand size, r/x/b for register extension bits.
func rex(w, r, x, b int) byte {
	return byte(0x40 | (w << 3) | (r << 2) | (x << 1) | b)
}

// rexFor returns the REX.B bit needed for a register (1 if r8-r15, 0 otherwise)
func rexB(reg int) int {
	if reg >= 8 {
		return 1
	}
	return 0
}

func rexR(reg int) int {
	if reg >= 8 {
		return 1
	}
	return 0
}

// regLow returns the low 3 bits of a register number
func regLow(reg int) byte {
	return byte(reg & 7)
}

// modRM builds a ModR/M byte
func modRM(mod, reg, rm int) byte {
	return byte((mod&3)<<6 | (reg&7)<<3 | (rm & 7))
}

// emitMovRegReg: mov dst, src (64-bit)
func (g *amd64Gen) emitMovRegReg(dst, src int) {
	g.emit(rex(1, rexR(src), 0, rexB(dst)))
	g.emit(0x89)
	g.emit(modRM(3, src, dst))
}

// emitMovRegImm32: mov reg, imm32 (32-bit, zero-extends to 64-bit)
func (g *amd64Gen) emitMovRegImm32(reg int, imm uint32) {
	if rexB(reg) != 0 {
		g.emit(rex(0, 0, 0, 1))
	}
	g.emit(0xB8 + regLow(reg))
	g.emit32le(imm)
}

// emitMovRegImm64: movabs reg, imm64
func (g *amd64Gen) emitMovRegImm64(reg int, imm uint64) {
	g.emit(rex(1, 0, 0, rexB(reg)))
	g.emit(0xB8 + regLow(reg))
	g.emit(byte(imm), byte(imm>>8), byte(imm>>16), byte(imm>>24),
		byte(imm>>32), byte(imm>>40), byte(imm>>48), byte(imm>>56))
}

// emitXorReg32: xor reg32, reg32
func (g *amd64Gen) emitXorReg32(reg int) {
	if rexB(reg) != 0 {
		g.emit(rex(0, rexR(reg), 0, rexB(reg)))
	}
	g.emit(0x31)
	g.emit(modRM(3, reg, reg))
}

// Memory addressing: [rbx + disp]
// Returns (mod, dispBytes) for ModR/M encoding
func dispMod(disp int) (int, []byte) {
	if disp == 0 {
		return 0, nil
	}
	if disp >= -128 && disp <= 127 {
		return 1, []byte{byte(int8(disp))}
	}
	b := make([]byte, 4)
	v := uint32(int32(disp))
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	return 2, b
}

// emitLoadMem loads from [rbx+disp] into reg with the appropriate width
func (g *amd64Gen) emitLoadMem(reg int, disp int) {
	mod, dispB := dispMod(disp)
	switch g.wordSize {
	case 8:
		// movzx reg32, byte [rbx+disp] — 0F B6 /r
		if rexR(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x0F, 0xB6)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 16:
		// movzx reg32, word [rbx+disp] — 0F B7 /r
		if rexR(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x0F, 0xB7)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 32:
		// mov reg32, [rbx+disp] — 8B /r
		if rexR(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x8B)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 64:
		// mov reg64, [rbx+disp] — REX.W 8B /r
		g.emit(rex(1, rexR(reg), 0, 0))
		g.emit(0x8B)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	}
}

// emitStoreMem stores reg to [rbx+disp] with the appropriate width
func (g *amd64Gen) emitStoreMem(reg int, disp int) {
	mod, dispB := dispMod(disp)
	switch g.wordSize {
	case 8:
		// mov byte [rbx+disp], reg8 — 88 /r
		// For reg >= 4 (RSP..RDI), need REX prefix to access SPL,BPL,SIL,DIL
		if rexR(reg) != 0 || reg >= 4 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x88)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 16:
		// mov word [rbx+disp], reg16 — 66 89 /r
		g.emit(0x66)
		if rexR(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x89)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 32:
		// mov dword [rbx+disp], reg32 — 89 /r
		if rexR(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, 0))
		}
		g.emit(0x89)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	case 64:
		// mov qword [rbx+disp], reg64 — REX.W 89 /r
		g.emit(rex(1, rexR(reg), 0, 0))
		g.emit(0x89)
		g.emit(modRM(mod, reg, RBX))
		g.emit(dispB...)
	}
}

// emitStoreZeroMem stores zero to [rbx+disp]
func (g *amd64Gen) emitStoreZeroMem(disp int) {
	mod, dispB := dispMod(disp)
	switch g.wordSize {
	case 8:
		// mov byte [rbx+disp], 0 — C6 /0 ib
		g.emit(0xC6)
		g.emit(modRM(mod, 0, RBX))
		g.emit(dispB...)
		g.emit(0)
	case 16:
		// mov word [rbx+disp], 0 — 66 C7 /0 iw
		g.emit(0x66, 0xC7)
		g.emit(modRM(mod, 0, RBX))
		g.emit(dispB...)
		g.emit(0, 0)
	case 32:
		// mov dword [rbx+disp], 0 — C7 /0 id
		g.emit(0xC7)
		g.emit(modRM(mod, 0, RBX))
		g.emit(dispB...)
		g.emit32le(0)
	case 64:
		// mov qword [rbx+disp], 0 — REX.W C7 /0 id (sign-extends imm32)
		g.emit(rex(1, 0, 0, 0))
		g.emit(0xC7)
		g.emit(modRM(mod, 0, RBX))
		g.emit(dispB...)
		g.emit32le(0)
	}
}

// emitLoadCell loads the cell at [rbx] into rax
func (g *amd64Gen) emitLoadCell() {
	g.emitLoadMem(RAX, 0)
}

// emitStoreCell stores rax to [rbx]
func (g *amd64Gen) emitStoreCell() {
	g.emitStoreMem(RAX, 0)
}

// emitLoadTo loads the cell at [rbx + offset*byteScale] into reg
func (g *amd64Gen) emitLoadTo(reg int, offset int) {
	g.emitLoadMem(reg, offset*g.byteScale)
}

// emitStoreTo stores reg to [rbx + offset*byteScale]
func (g *amd64Gen) emitStoreTo(reg int, offset int) {
	g.emitStoreMem(reg, offset*g.byteScale)
}

// emitMask masks rax to the current word size
func (g *amd64Gen) emitMask(reg int) {
	switch g.wordSize {
	case 8:
		// movzx eax, al — 0F B6 C0 (self zero-extend)
		if rexR(reg) != 0 || rexB(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, rexB(reg)))
		}
		g.emit(0x0F, 0xB6)
		g.emit(modRM(3, reg, reg))
	case 16:
		// movzx eax, ax — 0F B7 C0
		if rexR(reg) != 0 || rexB(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, rexB(reg)))
		}
		g.emit(0x0F, 0xB7)
		g.emit(modRM(3, reg, reg))
	}
	// 32/64 bit: no masking needed
}

// emitAddRegImm: add reg, imm32 (64-bit)
func (g *amd64Gen) emitAddRegImm(reg int, imm int) {
	if imm == 0 {
		return
	}
	if imm >= -128 && imm <= 127 {
		// add reg, imm8 — REX.W 83 /0 ib
		g.emit(rex(1, 0, 0, rexB(reg)))
		g.emit(0x83)
		g.emit(modRM(3, 0, reg))
		g.emit(byte(int8(imm)))
	} else {
		if reg == RAX {
			// add rax, imm32 — REX.W 05 id
			g.emit(rex(1, 0, 0, 0))
			g.emit(0x05)
		} else {
			// add reg, imm32 — REX.W 81 /0 id
			g.emit(rex(1, 0, 0, rexB(reg)))
			g.emit(0x81)
			g.emit(modRM(3, 0, reg))
		}
		g.emit32le(uint32(int32(imm)))
	}
}

// emitSubRegImm: sub reg, imm32 (64-bit)
func (g *amd64Gen) emitSubRegImm(reg int, imm int) {
	if imm == 0 {
		return
	}
	if imm >= -128 && imm <= 127 {
		g.emit(rex(1, 0, 0, rexB(reg)))
		g.emit(0x83)
		g.emit(modRM(3, 5, reg))
		g.emit(byte(int8(imm)))
	} else {
		if reg == RAX {
			g.emit(rex(1, 0, 0, 0))
			g.emit(0x2D)
		} else {
			g.emit(rex(1, 0, 0, rexB(reg)))
			g.emit(0x81)
			g.emit(modRM(3, 5, reg))
		}
		g.emit32le(uint32(int32(imm)))
	}
}

// emitAddCellImm adds imm to the cell value in reg (cell-width operation)
func (g *amd64Gen) emitAddCellImm(reg int, imm int) {
	if imm == 0 {
		return
	}
	switch g.wordSize {
	case 8:
		// add reg8, imm8
		if rexB(reg) != 0 || reg >= 4 {
			g.emit(rex(0, 0, 0, rexB(reg)))
		}
		g.emit(0x80)
		g.emit(modRM(3, 0, reg))
		g.emit(byte(int8(imm)))
	case 16:
		g.emit(0x66)
		if imm >= -128 && imm <= 127 {
			if rexB(reg) != 0 {
				g.emit(rex(0, 0, 0, rexB(reg)))
			}
			g.emit(0x83)
			g.emit(modRM(3, 0, reg))
			g.emit(byte(int8(imm)))
		} else {
			if reg == RAX {
				g.emit(0x05)
			} else {
				if rexB(reg) != 0 {
					g.emit(rex(0, 0, 0, rexB(reg)))
				}
				g.emit(0x81)
				g.emit(modRM(3, 0, reg))
			}
			g.emit(byte(imm), byte(imm>>8))
		}
	case 32:
		if imm >= -128 && imm <= 127 {
			if rexB(reg) != 0 {
				g.emit(rex(0, 0, 0, rexB(reg)))
			}
			g.emit(0x83)
			g.emit(modRM(3, 0, reg))
			g.emit(byte(int8(imm)))
		} else {
			if reg == RAX {
				g.emit(0x05)
			} else {
				if rexB(reg) != 0 {
					g.emit(rex(0, 0, 0, rexB(reg)))
				}
				g.emit(0x81)
				g.emit(modRM(3, 0, reg))
			}
			g.emit32le(uint32(int32(imm)))
		}
	case 64:
		g.emitAddRegImm(reg, imm)
	}
}

// emitSubCellImm subtracts imm from the cell value in reg (cell-width operation)
func (g *amd64Gen) emitSubCellImm(reg int, imm int) {
	if imm == 0 {
		return
	}
	switch g.wordSize {
	case 8:
		if rexB(reg) != 0 || reg >= 4 {
			g.emit(rex(0, 0, 0, rexB(reg)))
		}
		g.emit(0x80)
		g.emit(modRM(3, 5, reg))
		g.emit(byte(int8(imm)))
	case 16:
		g.emit(0x66)
		if imm >= -128 && imm <= 127 {
			if rexB(reg) != 0 {
				g.emit(rex(0, 0, 0, rexB(reg)))
			}
			g.emit(0x83)
			g.emit(modRM(3, 5, reg))
			g.emit(byte(int8(imm)))
		} else {
			if reg == RAX {
				g.emit(0x2D)
			} else {
				if rexB(reg) != 0 {
					g.emit(rex(0, 0, 0, rexB(reg)))
				}
				g.emit(0x81)
				g.emit(modRM(3, 5, reg))
			}
			g.emit(byte(imm), byte(imm>>8))
		}
	case 32:
		if imm >= -128 && imm <= 127 {
			if rexB(reg) != 0 {
				g.emit(rex(0, 0, 0, rexB(reg)))
			}
			g.emit(0x83)
			g.emit(modRM(3, 5, reg))
			g.emit(byte(int8(imm)))
		} else {
			if reg == RAX {
				g.emit(0x2D)
			} else {
				if rexB(reg) != 0 {
					g.emit(rex(0, 0, 0, rexB(reg)))
				}
				g.emit(0x81)
				g.emit(modRM(3, 5, reg))
			}
			g.emit32le(uint32(int32(imm)))
		}
	case 64:
		g.emitSubRegImm(reg, imm)
	}
}

// emitAddRegReg: add dst, src (cell-width)
func (g *amd64Gen) emitAddRegReg(dst, src int) {
	switch g.wordSize {
	case 8:
		if rexR(src) != 0 || rexB(dst) != 0 || src >= 4 || dst >= 4 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x00) // add r/m8, r8
		g.emit(modRM(3, src, dst))
	case 16:
		g.emit(0x66)
		if rexR(src) != 0 || rexB(dst) != 0 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x01)
		g.emit(modRM(3, src, dst))
	case 32:
		if rexR(src) != 0 || rexB(dst) != 0 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x01)
		g.emit(modRM(3, src, dst))
	case 64:
		g.emit(rex(1, rexR(src), 0, rexB(dst)))
		g.emit(0x01)
		g.emit(modRM(3, src, dst))
	}
}

// emitSubRegReg: sub dst, src (cell-width)
func (g *amd64Gen) emitSubRegReg(dst, src int) {
	switch g.wordSize {
	case 8:
		if rexR(src) != 0 || rexB(dst) != 0 || src >= 4 || dst >= 4 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x28)
		g.emit(modRM(3, src, dst))
	case 16:
		g.emit(0x66)
		if rexR(src) != 0 || rexB(dst) != 0 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x29)
		g.emit(modRM(3, src, dst))
	case 32:
		if rexR(src) != 0 || rexB(dst) != 0 {
			g.emit(rex(0, rexR(src), 0, rexB(dst)))
		}
		g.emit(0x29)
		g.emit(modRM(3, src, dst))
	case 64:
		g.emit(rex(1, rexR(src), 0, rexB(dst)))
		g.emit(0x29)
		g.emit(modRM(3, src, dst))
	}
}

// emitImul3: imul dst, src, imm32 (cell-width multiply)
func (g *amd64Gen) emitImul3(dst, src int, imm int) {
	switch g.wordSize {
	case 8, 16, 32:
		// For 8-bit we compute in 32-bit and mask later
		w := 0
		if g.wordSize == 16 {
			g.emit(0x66)
		}
		if rexR(dst) != 0 || rexB(src) != 0 {
			g.emit(rex(w, rexR(dst), 0, rexB(src)))
		}
		if g.wordSize == 16 {
			if imm >= -128 && imm <= 127 {
				g.emit(0x6B)
				g.emit(modRM(3, dst, src))
				g.emit(byte(int8(imm)))
			} else {
				g.emit(0x69)
				g.emit(modRM(3, dst, src))
				g.emit(byte(imm), byte(imm>>8))
			}
		} else {
			if imm >= -128 && imm <= 127 {
				g.emit(0x6B)
				g.emit(modRM(3, dst, src))
				g.emit(byte(int8(imm)))
			} else {
				g.emit(0x69)
				g.emit(modRM(3, dst, src))
				g.emit32le(uint32(int32(imm)))
			}
		}
	case 64:
		g.emit(rex(1, rexR(dst), 0, rexB(src)))
		if imm >= -128 && imm <= 127 {
			g.emit(0x6B)
			g.emit(modRM(3, dst, src))
			g.emit(byte(int8(imm)))
		} else {
			g.emit(0x69)
			g.emit(modRM(3, dst, src))
			g.emit32le(uint32(int32(imm)))
		}
	}
}

// emitDiv: unsigned divide rax by rcx. Result in rax.
// Clobbers rdx.
func (g *amd64Gen) emitDiv(divisorReg int) {
	// xor edx, edx
	g.emitXorReg32(RDX)
	switch g.wordSize {
	case 8:
		// For 8-bit, use 16-bit div: AX / r8 -> AL (quotient)
		// movzx eax, al first to ensure AH=0... actually AX = our value
		// We need: xor ah,ah then div r/m8 or use 32-bit div
		// Simpler: use 32-bit div since value is already zero-extended in eax
		if rexB(divisorReg) != 0 {
			g.emit(rex(0, 0, 0, rexB(divisorReg)))
		}
		g.emit(0xF7)
		g.emit(modRM(3, 6, divisorReg)) // div r/m32
	case 16:
		g.emit(0x66)
		if rexB(divisorReg) != 0 {
			g.emit(rex(0, 0, 0, rexB(divisorReg)))
		}
		g.emit(0xF7)
		g.emit(modRM(3, 6, divisorReg))
	case 32:
		if rexB(divisorReg) != 0 {
			g.emit(rex(0, 0, 0, rexB(divisorReg)))
		}
		g.emit(0xF7)
		g.emit(modRM(3, 6, divisorReg))
	case 64:
		g.emit(rex(1, 0, 0, rexB(divisorReg)))
		g.emit(0xF7)
		g.emit(modRM(3, 6, divisorReg))
	}
}

// emitTestRegReg: test reg, reg (cell-width)
func (g *amd64Gen) emitTestRegReg(reg int) {
	switch g.wordSize {
	case 8:
		if rexR(reg) != 0 || rexB(reg) != 0 || reg >= 4 {
			g.emit(rex(0, rexR(reg), 0, rexB(reg)))
		}
		g.emit(0x84)
		g.emit(modRM(3, reg, reg))
	case 16:
		g.emit(0x66)
		if rexR(reg) != 0 || rexB(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, rexB(reg)))
		}
		g.emit(0x85)
		g.emit(modRM(3, reg, reg))
	case 32:
		if rexR(reg) != 0 || rexB(reg) != 0 {
			g.emit(rex(0, rexR(reg), 0, rexB(reg)))
		}
		g.emit(0x85)
		g.emit(modRM(3, reg, reg))
	case 64:
		g.emit(rex(1, rexR(reg), 0, rexB(reg)))
		g.emit(0x85)
		g.emit(modRM(3, reg, reg))
	}
}

// emitJeRel32: JE rel32 — returns offset of the rel32 field for patching
func (g *amd64Gen) emitJeRel32() int {
	g.emit(0x0F, 0x84)
	off := g.pos()
	g.emit32le(0) // placeholder
	return off
}

// emitJmpRel32: JMP rel32 — returns offset of the rel32 field for patching
func (g *amd64Gen) emitJmpRel32() int {
	g.emit(0xE9)
	off := g.pos()
	g.emit32le(0) // placeholder
	return off
}

// emitSyscall: syscall instruction
func (g *amd64Gen) emitSyscall() {
	g.emit(0x0F, 0x05)
}

// emitCallReg: call reg
func (g *amd64Gen) emitCallReg(reg int) {
	if rexB(reg) != 0 {
		g.emit(rex(0, 0, 0, 1))
	}
	g.emit(0xFF)
	g.emit(modRM(3, 2, reg))
}

// emitRet: ret
func (g *amd64Gen) emitRet() {
	g.emit(0xC3)
}

// emitPush: push reg (64-bit)
func (g *amd64Gen) emitPush(reg int) {
	if rexB(reg) != 0 {
		g.emit(rex(0, 0, 0, 1))
	}
	g.emit(0x50 + regLow(reg))
}

// emitPop: pop reg (64-bit)
func (g *amd64Gen) emitPop(reg int) {
	if rexB(reg) != 0 {
		g.emit(rex(0, 0, 0, 1))
	}
	g.emit(0x58 + regLow(reg))
}

// emitMovImm loads an immediate into reg (choosing smallest encoding)
func (g *amd64Gen) emitMovImm(reg int, value int) {
	if value == 0 {
		g.emitXorReg32(reg)
	} else if value > 0 && value <= 0xFFFFFFFF {
		g.emitMovRegImm32(reg, uint32(value))
	} else {
		g.emitMovRegImm64(reg, uint64(int64(value)))
	}
}

// emitMovzxByte: movzx reg32, reg8  (zero-extend byte to 32-bit)
func (g *amd64Gen) emitMovzxByte(dst, src int) {
	if rexR(dst) != 0 || rexB(src) != 0 || src >= 4 {
		g.emit(rex(0, rexR(dst), 0, rexB(src)))
	}
	g.emit(0x0F, 0xB6)
	g.emit(modRM(3, dst, src))
}
