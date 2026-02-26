package generators

// ARM64 register constants
const (
	X0  = 0
	X1  = 1
	X2  = 2
	X9  = 9
	X10 = 10
	X11 = 11
	X19 = 19
	X20 = 20
	X21 = 21
	X22 = 22
	X29 = 29
	X30 = 30
	XZR = 31
	SP  = 31
)

// encAddImm: ADD Rd, Rn, #imm12 (sf=1 for 64-bit, sf=0 for 32-bit)
func encAddImm(sf, rd, rn int, imm uint32, shift int) uint32 {
	return uint32(sf)<<31 | 0b00100010<<23 | uint32(shift)<<22 |
		(imm&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// encSubImm: SUB Rd, Rn, #imm12
func encSubImm(sf, rd, rn int, imm uint32, shift int) uint32 {
	return uint32(sf)<<31 | 0b10100010<<23 | uint32(shift)<<22 |
		(imm&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// encAddReg: ADD Rd, Rn, Rm
func encAddReg(sf, rd, rn, rm int) uint32 {
	return uint32(sf)<<31 | 0b0001011000<<21 | uint32(rm&0x1F)<<16 |
		uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// encSubReg: SUB Rd, Rn, Rm
func encSubReg(sf, rd, rn, rm int) uint32 {
	return uint32(sf)<<31 | 0b1001011000<<21 | uint32(rm&0x1F)<<16 |
		uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// encMovz: MOVZ Rd, #imm16, LSL #(hw*16)
func encMovz(sf, rd int, imm16 uint32, hw int) uint32 {
	return uint32(sf)<<31 | 0b10100101<<23 | uint32(hw)<<21 |
		(imm16&0xFFFF)<<5 | uint32(rd&0x1F)
}

// encMovk: MOVK Rd, #imm16, LSL #(hw*16)
func encMovk(sf, rd int, imm16 uint32, hw int) uint32 {
	return uint32(sf)<<31 | 0b11100101<<23 | uint32(hw)<<21 |
		(imm16&0xFFFF)<<5 | uint32(rd&0x1F)
}

// encMovReg: MOV Rd, Rn — alias for ORR Rd, XZR, Rn
func encMovReg(sf, rd, rm int) uint32 {
	return uint32(sf)<<31 | 0b01010100<<23 | 0<<22 | 0<<21 | uint32(rm&0x1F)<<16 |
		0<<10 | uint32(XZR&0x1F)<<5 | uint32(rd&0x1F)
}

// Load/Store unsigned offset
// LDRB Wt, [Xn, #imm12]
func encLdrbUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b0011100101<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STRB Wt, [Xn, #imm12]
func encStrbUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b0011100100<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDRH Wt, [Xn, #imm12*2] — imm12 is byte_offset/2
func encLdrhUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b0111100101<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STRH Wt, [Xn, #imm12*2]
func encStrhUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b0111100100<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDR Wt, [Xn, #imm12*4] — imm12 is byte_offset/4
func encLdrwUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b1011100101<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STR Wt, [Xn, #imm12*4]
func encStrwUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b1011100100<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDR Xt, [Xn, #imm12*8] — imm12 is byte_offset/8
func encLdrxUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b1111100101<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STR Xt, [Xn, #imm12*8]
func encStrxUoff(rt, rn int, imm12 uint32) uint32 {
	return 0b1111100100<<22 | (imm12&0xFFF)<<10 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// Load/Store unscaled (signed 9-bit offset)
// LDURB Wt, [Xn, #simm9]
func encLdurb(rt, rn int, simm9 int) uint32 {
	return 0b00111000010<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STURB Wt, [Xn, #simm9]
func encSturb(rt, rn int, simm9 int) uint32 {
	return 0b00111000000<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDURH Wt, [Xn, #simm9]
func encLdurh(rt, rn int, simm9 int) uint32 {
	return 0b01111000010<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STURH Wt, [Xn, #simm9]
func encSturh(rt, rn int, simm9 int) uint32 {
	return 0b01111000000<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDUR Wt, [Xn, #simm9]
func encLdurw(rt, rn int, simm9 int) uint32 {
	return 0b10111000010<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STUR Wt, [Xn, #simm9]
func encSturw(rt, rn int, simm9 int) uint32 {
	return 0b10111000000<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// LDUR Xt, [Xn, #simm9]
func encLdurx(rt, rn int, simm9 int) uint32 {
	return 0b11111000010<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// STUR Xt, [Xn, #simm9]
func encSturx(rt, rn int, simm9 int) uint32 {
	return 0b11111000000<<21 | (uint32(simm9)&0x1FF)<<12 | uint32(rn&0x1F)<<5 | uint32(rt&0x1F)
}

// Branch instructions
// CBZ Rt, #offset (offset in instructions)
func encCbz(sf, rt int, offsetInstr int) uint32 {
	return uint32(sf)<<31 | 0b0110100<<24 | (uint32(offsetInstr)&0x7FFFF)<<5 | uint32(rt&0x1F)
}

// CBNZ Rt, #offset
func encCbnz(sf, rt int, offsetInstr int) uint32 {
	return uint32(sf)<<31 | 0b0110101<<24 | (uint32(offsetInstr)&0x7FFFF)<<5 | uint32(rt&0x1F)
}

// B #offset (unconditional, offset in instructions)
func encB(offsetInstr int) uint32 {
	return 0b000101<<26 | (uint32(offsetInstr) & 0x3FFFFFF)
}

// BLR Xn
func encBlr(rn int) uint32 {
	return 0xD63F0000 | uint32(rn&0x1F)<<5
}

// RET (via x30)
func encRet() uint32 {
	return 0xD65F03C0
}

// Store/Load pair (64-bit registers)
// STP Xt1, Xt2, [Xn, #imm7*8]! (pre-index)
func encStpPre64(rt1, rt2, rn int, imm7 int) uint32 {
	return 0b1010100110<<22 | (uint32(imm7)&0x7F)<<15 |
		uint32(rt2&0x1F)<<10 | uint32(rn&0x1F)<<5 | uint32(rt1&0x1F)
}

// LDP Xt1, Xt2, [Xn], #imm7*8 (post-index)
func encLdpPost64(rt1, rt2, rn int, imm7 int) uint32 {
	return 0b1010100011<<22 | (uint32(imm7)&0x7F)<<15 |
		uint32(rt2&0x1F)<<10 | uint32(rn&0x1F)<<5 | uint32(rt1&0x1F)
}

// STP Xt1, Xt2, [Xn, #imm7*8] (signed offset, no writeback)
func encStpOff64(rt1, rt2, rn int, imm7 int) uint32 {
	return 0b1010100100<<22 | (uint32(imm7)&0x7F)<<15 |
		uint32(rt2&0x1F)<<10 | uint32(rn&0x1F)<<5 | uint32(rt1&0x1F)
}

// LDP Xt1, Xt2, [Xn, #imm7*8] (signed offset, no writeback)
func encLdpOff64(rt1, rt2, rn int, imm7 int) uint32 {
	return 0b1010100101<<22 | (uint32(imm7)&0x7F)<<15 |
		uint32(rt2&0x1F)<<10 | uint32(rn&0x1F)<<5 | uint32(rt1&0x1F)
}

// Multiply/Divide
// MADD Rd, Rn, Rm, Ra (Rd = Ra + Rn * Rm)
func encMadd(sf, rd, rn, rm, ra int) uint32 {
	return uint32(sf)<<31 | 0b0011011000<<21 | uint32(rm&0x1F)<<16 |
		uint32(ra&0x1F)<<10 | uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// UDIV Rd, Rn, Rm
func encUdiv(sf, rd, rn, rm int) uint32 {
	return uint32(sf)<<31 | 0b0011010110<<21 | uint32(rm&0x1F)<<16 |
		0b000010<<10 | uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// AND immediate — uses the logical immediate encoding (N:immr:imms)
func encAndImm(sf, rd, rn int, n, immr, imms int) uint32 {
	return uint32(sf)<<31 | 0b00100100<<23 | uint32(n)<<22 |
		uint32(immr&0x3F)<<16 | uint32(imms&0x3F)<<10 | uint32(rn&0x1F)<<5 | uint32(rd&0x1F)
}

// AND Wd, Wn, #0xFF (8-bit mask)
func encAndImm8(rd, rn int) uint32 {
	return encAndImm(0, rd, rn, 0, 0, 7)
}

// AND Wd, Wn, #0xFFFF (16-bit mask)
func encAndImm16(rd, rn int) uint32 {
	return encAndImm(0, rd, rn, 0, 0, 15)
}

// Higher-level helpers

// emitMovImm appends instructions to load an immediate into a register.
// Uses MOVZ + MOVK as needed.
func emitMovImm(code *[]uint32, sf, rd int, value uint64) {
	*code = append(*code, encMovz(sf, rd, uint32(value&0xFFFF), 0))
	if value > 0xFFFF {
		*code = append(*code, encMovk(sf, rd, uint32((value>>16)&0xFFFF), 1))
	}
	if value > 0xFFFFFFFF {
		*code = append(*code, encMovk(sf, rd, uint32((value>>32)&0xFFFF), 2))
	}
	if value > 0xFFFFFFFFFFFF {
		*code = append(*code, encMovk(sf, rd, uint32((value>>48)&0xFFFF), 3))
	}
}

// emitAddImm appends ADD instruction(s) for an arbitrary positive immediate.
func emitAddImm(code *[]uint32, sf, rd, rn int, imm int, scratch int) {
	if imm <= 0xFFF {
		*code = append(*code, encAddImm(sf, rd, rn, uint32(imm), 0))
	} else if imm <= 0xFFF<<12 && imm&0xFFF == 0 {
		*code = append(*code, encAddImm(sf, rd, rn, uint32(imm>>12), 1))
	} else {
		emitMovImm(code, 1, scratch, uint64(imm))
		*code = append(*code, encAddReg(sf, rd, rn, scratch))
	}
}

// emitSubImm appends SUB instruction(s) for an arbitrary positive immediate.
func emitSubImm(code *[]uint32, sf, rd, rn int, imm int, scratch int) {
	if imm <= 0xFFF {
		*code = append(*code, encSubImm(sf, rd, rn, uint32(imm), 0))
	} else if imm <= 0xFFF<<12 && imm&0xFFF == 0 {
		*code = append(*code, encSubImm(sf, rd, rn, uint32(imm>>12), 1))
	} else {
		emitMovImm(code, 1, scratch, uint64(imm))
		*code = append(*code, encSubReg(sf, rd, rn, scratch))
	}
}
