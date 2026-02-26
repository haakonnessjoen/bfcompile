package generators

import (
	l "bcomp/lexer"
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

// Mach-O constants
const (
	MH_MAGIC_64    = 0xFEEDFACF
	CPU_TYPE_ARM64 = 0x0100000C
	CPU_SUBTYPE_ALL = 0x00000000
	MH_EXECUTE     = 0x2
	MH_PIE         = 0x00200000
	MH_DYLDLINK    = 0x4
	LC_SEGMENT_64  = 0x19
	LC_MAIN        = 0x80000028
	LC_LOAD_DYLINKER = 0xE
	LC_SYMTAB      = 0x2
	LC_DYSYMTAB    = 0xB

	VM_PROT_READ    = 0x1
	VM_PROT_WRITE   = 0x2
	VM_PROT_EXECUTE = 0x4

	S_REGULAR  = 0x0
	S_ZEROFILL = 0x1

	S_ATTR_PURE_INSTRUCTIONS = 0x80000000
	S_ATTR_SOME_INSTRUCTIONS = 0x00000400

	pageSize = 0x4000
)

// machoWriter is a helper for building a little-endian binary blob.
type machoWriter struct {
	buf []byte
}

func (w *machoWriter) u32(v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	w.buf = append(w.buf, b...)
}

func (w *machoWriter) u64(v uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	w.buf = append(w.buf, b...)
}

func (w *machoWriter) strpad(s string, n int) {
	b := make([]byte, n)
	copy(b, s)
	w.buf = append(w.buf, b...)
}

func (w *machoWriter) pad(n int) {
	w.buf = append(w.buf, make([]byte, n)...)
}

func (w *machoWriter) pos() int {
	return len(w.buf)
}

func alignUp(v, align uint64) uint64 {
	return (v + align - 1) &^ (align - 1)
}

// PrintARM64 generates a Mach-O ARM64 binary directly.
func PrintARM64(outpath string, tokens []ParseToken, memorySize int, wordSize int) {
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

	// --- Prologue: 2 instructions (ADRP + ADD), patched later ---
	gen.emit(0) // placeholder for ADRP x19, <page>
	gen.emit(0) // placeholder for ADD x19, x19, #<off>

	// --- Token code generation ---
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
				gen.emit(encMovz(0, X0, 1, 0))        // mov w0, #1 (stdout)
				gen.emit(encMovReg(1, X1, X19))        // mov x1, x19 (buf)
				gen.emit(encMovz(0, X2, 1, 0))         // mov w2, #1 (len)
				gen.emit(encMovz(0, X16, 4, 0))        // mov w16, #4 (write)
				gen.emit(encSvc(0x80))                  // svc #0x80
			}

		case l.IN:
			for i := 0; i < t.Extra; i++ {
				// Zero cell first
				switch gen.wordSize {
				case 8:
					gen.emit(encStrbUoff(XZR, X19, 0))
				case 16:
					gen.emit(encStrhUoff(XZR, X19, 0))
				case 32:
					gen.emit(encStrwUoff(XZR, X19, 0))
				case 64:
					gen.emit(encStrxUoff(XZR, X19, 0))
				}
				gen.emit(encMovz(0, X0, 0, 0))         // mov w0, #0 (stdin)
				gen.emit(encMovReg(1, X1, X19))         // mov x1, x19 (buf)
				gen.emit(encMovz(0, X2, 1, 0))          // mov w2, #1 (len)
				gen.emit(encMovz(0, X16, 3, 0))         // mov w16, #3 (read)
				gen.emit(encSvc(0x80))                   // svc #0x80
			}

		case l.JMPF:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0)) // placeholder
			gen.loopStack = append(gen.loopStack, loopInfo{topIdx, cbzIdx})

		case l.JMPB:
			info := gen.loopStack[len(gen.loopStack)-1]
			gen.loopStack = gen.loopStack[:len(gen.loopStack)-1]
			gen.emit(encB(info.topIdx - gen.pos()))
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
			gen.emitLoadCell()
			if multiplier == 1 {
				gen.emitLoadTo(X10, offset)
				gen.emit(encAddReg(gen.sf, X10, X10, X9))
			} else if multiplier == -1 {
				gen.emitLoadTo(X10, offset)
				gen.emit(encSubReg(gen.sf, X10, X10, X9))
			} else {
				abs := multiplier
				if abs < 0 {
					abs = -abs
				}
				emitMovImm(&gen.code, gen.sf, X11, uint64(abs))
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
			gen.emit(encCbz(gen.sf, X9, 0))
			gen.emit(encAddImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		case l.SCANL:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0))
			gen.emit(encSubImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		case l.PRNT:
			topIdx := gen.pos()
			gen.emitLoadCell()
			cbzIdx := gen.pos()
			gen.emit(encCbz(gen.sf, X9, 0))
			gen.emit(encMovz(0, X0, 1, 0))        // mov w0, #1
			gen.emit(encMovReg(1, X1, X19))        // mov x1, x19
			gen.emit(encMovz(0, X2, 1, 0))         // mov w2, #1
			gen.emit(encMovz(0, X16, 4, 0))        // mov w16, #4
			gen.emit(encSvc(0x80))                  // svc #0x80
			gen.emit(encAddImm(1, X19, X19, uint32(gen.byteScale), 0))
			gen.emit(encB(topIdx - gen.pos()))
			gen.code[cbzIdx] = encCbz(gen.sf, X9, gen.pos()-cbzIdx)

		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}

	// --- Epilogue: exit(0) syscall ---
	gen.emit(encMovz(0, X0, 0, 0))   // mov w0, #0
	gen.emit(encMovz(0, X16, 1, 0))  // mov w16, #1 (exit)
	gen.emit(encSvc(0x80))            // svc #0x80

	// --- Layout computation ---
	numInstr := len(gen.code)
	codeBytes := numInstr * 4

	// Header + load commands = 640 bytes
	const headerSize = 32
	const numCmds = 8
	// Sizes: PAGEZERO=72, TEXT=152, DATA=152, LINKEDIT=72, MAIN=24, DYLINKER=32, SYMTAB=24, DYSYMTAB=80
	const cmdsSize = 72 + 152 + 152 + 72 + 24 + 32 + 24 + 80 // = 608
	// Leave slack after load commands for codesign to add LC_CODE_SIGNATURE (16 bytes)
	const codesignSlack = 32
	const textSectionOff = headerSize + cmdsSize + codesignSlack // 672

	textFileSize := uint64(textSectionOff) + uint64(codeBytes)
	textVMSize := alignUp(textFileSize, pageSize)
	textFileTotal := textVMSize // pad file to page boundary

	dataVMAddr := 0x100000000 + textVMSize
	bssSize := uint64(memorySize) * uint64(wordSize/8)
	dataVMSize := alignUp(bssSize, pageSize)

	linkeditVMAddr := dataVMAddr + dataVMSize

	// --- Patch ADRP + ADD ---
	adrpPC := uint64(0x100000000) + uint64(textSectionOff) + 0
	pageOff := int(dataVMAddr>>12) - int(adrpPC>>12)
	lowOff := uint32(dataVMAddr & 0xFFF) // will be 0 since data is page-aligned
	gen.code[0] = encAdrp(X19, pageOff)
	gen.code[1] = encAddImm(1, X19, X19, lowOff, 0)

	// --- Build binary ---
	w := &machoWriter{buf: make([]byte, 0, int(textFileTotal)+pageSize)}

	// mach_header_64
	w.u32(MH_MAGIC_64)
	w.u32(CPU_TYPE_ARM64)
	w.u32(CPU_SUBTYPE_ALL)
	w.u32(MH_EXECUTE)
	w.u32(numCmds)
	w.u32(cmdsSize)
	w.u32(MH_PIE | MH_DYLDLINK)
	w.u32(0) // reserved

	// LC_SEGMENT_64 __PAGEZERO
	w.u32(LC_SEGMENT_64)
	w.u32(72) // cmdsize
	w.strpad("__PAGEZERO", 16)
	w.u64(0)           // vmaddr
	w.u64(0x100000000) // vmsize (4GB)
	w.u64(0)           // fileoff
	w.u64(0)           // filesize
	w.u32(0)           // maxprot
	w.u32(0)           // initprot
	w.u32(0)           // nsects
	w.u32(0)           // flags

	// LC_SEGMENT_64 __TEXT + 1 section
	w.u32(LC_SEGMENT_64)
	w.u32(152) // cmdsize (72 + 80 for 1 section)
	w.strpad("__TEXT", 16)
	w.u64(0x100000000) // vmaddr
	w.u64(textVMSize)  // vmsize
	w.u64(0)           // fileoff
	w.u64(textFileTotal) // filesize (padded to page)
	w.u32(VM_PROT_READ | VM_PROT_EXECUTE) // maxprot
	w.u32(VM_PROT_READ | VM_PROT_EXECUTE) // initprot
	w.u32(1) // nsects
	w.u32(0) // flags

	// Section: __text
	w.strpad("__text", 16)        // sectname
	w.strpad("__TEXT", 16)        // segname
	w.u64(0x100000000 + uint64(textSectionOff)) // addr
	w.u64(uint64(codeBytes))      // size
	w.u32(uint32(textSectionOff)) // offset
	w.u32(2)                      // align (2^2 = 4)
	w.u32(0)                      // reloff
	w.u32(0)                      // nreloc
	w.u32(S_REGULAR | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS) // flags
	w.u32(0) // reserved1
	w.u32(0) // reserved2
	w.u32(0) // reserved3

	// LC_SEGMENT_64 __DATA + 1 section (__bss, S_ZEROFILL)
	w.u32(LC_SEGMENT_64)
	w.u32(152)
	w.strpad("__DATA", 16)
	w.u64(dataVMAddr)  // vmaddr
	w.u64(dataVMSize)  // vmsize
	w.u64(0)           // fileoff (no file backing)
	w.u64(0)           // filesize
	w.u32(VM_PROT_READ | VM_PROT_WRITE) // maxprot
	w.u32(VM_PROT_READ | VM_PROT_WRITE) // initprot
	w.u32(1) // nsects
	w.u32(0) // flags

	// Section: __bss
	w.strpad("__bss", 16)    // sectname
	w.strpad("__DATA", 16)   // segname
	w.u64(dataVMAddr)        // addr
	w.u64(bssSize)            // size
	w.u32(0)                  // offset (no file data)
	w.u32(0)                  // align
	w.u32(0)                  // reloff
	w.u32(0)                  // nreloc
	w.u32(S_ZEROFILL)         // flags
	w.u32(0) // reserved1
	w.u32(0) // reserved2
	w.u32(0) // reserved3

	// LC_SEGMENT_64 __LINKEDIT
	w.u32(LC_SEGMENT_64)
	w.u32(72)
	w.strpad("__LINKEDIT", 16)
	w.u64(linkeditVMAddr) // vmaddr
	w.u64(uint64(pageSize))        // vmsize (1 page minimum)
	w.u64(textFileTotal)  // fileoff (at end of __TEXT file data)
	w.u64(0)              // filesize (empty)
	w.u32(VM_PROT_READ)   // maxprot
	w.u32(VM_PROT_READ)   // initprot
	w.u32(0) // nsects
	w.u32(0) // flags

	// LC_MAIN
	w.u32(LC_MAIN)
	w.u32(24)                             // cmdsize
	w.u64(uint64(textSectionOff))         // entryoff (offset from __TEXT start)
	w.u64(0)                              // stacksize (default)

	// LC_LOAD_DYLINKER
	dylinkerPath := "/usr/lib/dyld"
	dylinkerCmdSize := uint32(alignUp(uint64(12+len(dylinkerPath)+1), 4)) // lc header=12, string, null, align
	// But spec says cmdsize must be multiple of 8 for 64-bit
	dylinkerCmdSize = uint32(alignUp(uint64(dylinkerCmdSize), 8))
	w.u32(LC_LOAD_DYLINKER)
	w.u32(dylinkerCmdSize)
	w.u32(12) // name offset (right after cmd, cmdsize, offset)
	w.strpad(dylinkerPath, int(dylinkerCmdSize)-12)

	// LC_SYMTAB
	w.u32(LC_SYMTAB)
	w.u32(24)
	w.u32(0) // symoff
	w.u32(0) // nsyms
	w.u32(0) // stroff
	w.u32(0) // strsize

	// LC_DYSYMTAB
	w.u32(LC_DYSYMTAB)
	w.u32(80)
	w.pad(80 - 8) // all fields zero

	// Pad to textSectionOff (slack space for codesign to add LC_CODE_SIGNATURE)
	if pad := textSectionOff - w.pos(); pad > 0 {
		w.pad(pad)
	}
	if w.pos() != textSectionOff {
		log.Fatalf("Header+commands+padding = %d bytes, expected %d", w.pos(), textSectionOff)
	}

	// Write code
	for _, instr := range gen.code {
		w.u32(instr)
	}

	// Pad to page boundary
	padLen := int(textFileTotal) - w.pos()
	if padLen > 0 {
		w.pad(padLen)
	}

	// Write file
	if outpath == "" || outpath == "-" {
		if _, err := os.Stdout.Write(w.buf); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to stdout: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := os.WriteFile(outpath, w.buf, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s (%d ARM64 instructions)\n", len(w.buf), outpath, numInstr)
		fmt.Fprintf(os.Stderr, "Run: codesign --sign - %s\n", outpath)
	}
}
