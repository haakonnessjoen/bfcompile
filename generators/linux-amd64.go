package generators

import (
	l "bcomp/lexer"
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

// ELF constants
const (
	elfMagic    = "\x7fELF"
	elfClass64  = 2
	elfData2LSB = 1
	elfVersion1 = 1
	elfOSABI    = 0 // ELFOSABI_NONE (System V)

	etExec = 2 // ET_EXEC
	emX8664 = 62 // EM_X86_64

	ptLoad = 1 // PT_LOAD
	pfX    = 1
	pfW    = 2
	pfR    = 4

	elfPageSize = 0x1000
)

// elfWriter is a helper for building a little-endian binary blob.
type elfWriter struct {
	buf []byte
}

func (w *elfWriter) u8(v byte)    { w.buf = append(w.buf, v) }
func (w *elfWriter) u16(v uint16) { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); w.buf = append(w.buf, b...) }
func (w *elfWriter) u32(v uint32) { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); w.buf = append(w.buf, b...) }
func (w *elfWriter) u64(v uint64) { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); w.buf = append(w.buf, b...) }
func (w *elfWriter) pad(n int)    { w.buf = append(w.buf, make([]byte, n)...) }
func (w *elfWriter) pos() int     { return len(w.buf) }
func (w *elfWriter) bytes(b []byte) { w.buf = append(w.buf, b...) }

// PrintLinuxAMD64 generates a minimal static ELF x86-64 binary.
func PrintLinuxAMD64(outpath string, tokens []ParseToken, memorySize int, wordSize int) {
	gen := &amd64Gen{
		code:      make([]byte, 0, 4096),
		wordSize:  wordSize,
		byteScale: wordSize / 8,
		loopStack: make([]amd64LoopInfo, 0, 32),
		ifPatch:   make(map[int]int),
	}

	// ELF header: 64 bytes, 2 program headers: 2*56=112 bytes
	// Total header: 64 + 112 = 176 bytes
	const elfHeaderSize = 64
	const phdrSize = 56
	const numPhdrs = 2
	const headerTotal = elfHeaderSize + phdrSize*numPhdrs // 176

	const baseAddr = 0x400000

	// Prologue: mov rbx, <bss_vaddr> (10 bytes, patched later)
	prologueOff := gen.pos()
	gen.emit(rex(1, 0, 0, 0))             // REX.W
	gen.emit(0xB8 + regLow(RBX))          // MOV rbx, imm64
	gen.emit(0, 0, 0, 0, 0, 0, 0, 0)     // placeholder imm64

	// Token code generation
	for _, t := range tokens {
		switch t.Tok.Tok {
		case l.ADD:
			gen.emitLoadCell()
			gen.emitAddCellImm(RAX, t.Extra)
			gen.emitMask(RAX)
			gen.emitStoreCell()

		case l.SUB:
			gen.emitLoadCell()
			gen.emitSubCellImm(RAX, t.Extra)
			gen.emitMask(RAX)
			gen.emitStoreCell()

		case l.INCP:
			gen.emitAddRegImm(RBX, t.Extra*gen.byteScale)

		case l.DECP:
			gen.emitSubRegImm(RBX, t.Extra*gen.byteScale)

		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				// write(1, rbx, 1)
				gen.emitMovRegImm32(RAX, 1)       // syscall number: write
				gen.emitMovRegImm32(RDI, 1)       // fd: stdout
				gen.emitMovRegReg(RSI, RBX)       // buf: memory pointer
				gen.emitMovRegImm32(RDX, 1)       // count: 1
				gen.emitSyscall()
			}

		case l.IN:
			for i := 0; i < t.Extra; i++ {
				// Zero the cell first
				gen.emitStoreZeroMem(0)
				// read(0, rbx, 1)
				gen.emitXorReg32(RAX)              // syscall number: read = 0
				gen.emitXorReg32(RDI)              // fd: stdin = 0
				gen.emitMovRegReg(RSI, RBX)        // buf: memory pointer
				gen.emitMovRegImm32(RDX, 1)        // count: 1
				gen.emitSyscall()
			}

		case l.JMPF:
			topOff := gen.pos()
			gen.emitLoadCell()
			gen.emitTestRegReg(RAX)
			jeOff := gen.emitJeRel32()
			gen.loopStack = append(gen.loopStack, amd64LoopInfo{topOff, jeOff})

		case l.JMPB:
			info := gen.loopStack[len(gen.loopStack)-1]
			gen.loopStack = gen.loopStack[:len(gen.loopStack)-1]
			// JMP back to top
			jmpOff := gen.emitJmpRel32()
			gen.patch32(jmpOff, uint32(int32(info.topOff-gen.pos())))
			// Patch the forward JE
			gen.patch32(info.jeOff, uint32(int32(gen.pos()-(info.jeOff+4))))

		case l.BZ:
			gen.emitLoadCell()
			gen.emitTestRegReg(RAX)
			jeOff := gen.emitJeRel32()
			gen.ifPatch[t.Extra] = jeOff

		case l.LBL:
			jeOff := gen.ifPatch[t.Extra]
			gen.patch32(jeOff, uint32(int32(gen.pos()-(jeOff+4))))

		case l.MUL:
			multiplier := t.Extra
			offset := t.Extra2
			gen.emitLoadCell() // rax = *p
			if multiplier == 1 {
				gen.emitLoadTo(RCX, offset)
				gen.emitAddRegReg(RCX, RAX)
			} else if multiplier == -1 {
				gen.emitLoadTo(RCX, offset)
				gen.emitSubRegReg(RCX, RAX)
			} else {
				abs := multiplier
				if abs < 0 {
					abs = -abs
				}
				gen.emitImul3(RCX, RAX, abs)
				gen.emitLoadTo(RAX, offset) // reuse rax
				if multiplier > 0 {
					gen.emitAddRegReg(RAX, RCX)
				} else {
					gen.emitSubRegReg(RAX, RCX)
				}
				// result is in rax, store from rax
				gen.emitMask(RAX)
				gen.emitStoreTo(RAX, offset)
				continue
			}
			gen.emitMask(RCX)
			gen.emitStoreTo(RCX, offset)

		case l.DIV:
			divisor := t.Extra
			offset := t.Extra2
			gen.emitLoadTo(RAX, offset)
			gen.emitMovImm(RCX, divisor)
			gen.emitDiv(RCX)
			gen.emitMask(RAX)
			gen.emitStoreTo(RAX, offset)

		case l.MOV:
			value := t.Extra
			offset := t.Extra2
			if value == 0 {
				gen.emitStoreZeroMem(offset * gen.byteScale)
			} else {
				gen.emitMovImm(RAX, value)
				gen.emitMask(RAX)
				gen.emitStoreTo(RAX, offset)
			}

		case l.SCANR:
			topOff := gen.pos()
			gen.emitLoadCell()
			gen.emitTestRegReg(RAX)
			jeOff := gen.emitJeRel32()
			gen.emitAddRegImm(RBX, gen.byteScale)
			jmpOff := gen.emitJmpRel32()
			gen.patch32(jmpOff, uint32(int32(topOff-gen.pos())))
			gen.patch32(jeOff, uint32(int32(gen.pos()-(jeOff+4))))

		case l.SCANL:
			topOff := gen.pos()
			gen.emitLoadCell()
			gen.emitTestRegReg(RAX)
			jeOff := gen.emitJeRel32()
			gen.emitSubRegImm(RBX, gen.byteScale)
			jmpOff := gen.emitJmpRel32()
			gen.patch32(jmpOff, uint32(int32(topOff-gen.pos())))
			gen.patch32(jeOff, uint32(int32(gen.pos()-(jeOff+4))))

		case l.PRNT:
			topOff := gen.pos()
			gen.emitLoadCell()
			gen.emitTestRegReg(RAX)
			jeOff := gen.emitJeRel32()
			// write(1, rbx, 1)
			gen.emitMovRegImm32(RAX, 1)
			gen.emitMovRegImm32(RDI, 1)
			gen.emitMovRegReg(RSI, RBX)
			gen.emitMovRegImm32(RDX, 1)
			gen.emitSyscall()
			gen.emitAddRegImm(RBX, gen.byteScale)
			jmpOff := gen.emitJmpRel32()
			gen.patch32(jmpOff, uint32(int32(topOff-gen.pos())))
			gen.patch32(jeOff, uint32(int32(gen.pos()-(jeOff+4))))

		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}

	// Epilogue: exit(0)
	gen.emitMovRegImm32(RAX, 60) // syscall number: exit
	gen.emitXorReg32(RDI)        // status: 0
	gen.emitSyscall()

	// Layout computation
	codeSize := len(gen.code)
	textFileSize := uint64(headerTotal) + uint64(codeSize)
	textVMSize := (textFileSize + elfPageSize - 1) &^ (elfPageSize - 1)

	bssVAddr := baseAddr + textVMSize
	bssSize := uint64(memorySize) * uint64(gen.byteScale)
	bssVMSize := (bssSize + elfPageSize - 1) &^ (elfPageSize - 1)

	// Patch prologue: mov rbx, bssVAddr
	binary.LittleEndian.PutUint64(gen.code[prologueOff+2:prologueOff+10], bssVAddr)

	// Build ELF
	w := &elfWriter{buf: make([]byte, 0, int(textVMSize))}

	// ELF header
	w.bytes([]byte(elfMagic))        // e_ident[0..3]: magic
	w.u8(elfClass64)                 // e_ident[4]: class
	w.u8(elfData2LSB)                // e_ident[5]: data
	w.u8(elfVersion1)                // e_ident[6]: version
	w.u8(elfOSABI)                   // e_ident[7]: OS/ABI
	w.pad(8)                         // e_ident[8..15]: padding
	w.u16(etExec)                    // e_type
	w.u16(emX8664)                   // e_machine
	w.u32(1)                         // e_version
	w.u64(baseAddr + headerTotal)    // e_entry
	w.u64(elfHeaderSize)             // e_phoff
	w.u64(0)                         // e_shoff (no section headers)
	w.u32(0)                         // e_flags
	w.u16(elfHeaderSize)             // e_ehsize
	w.u16(phdrSize)                  // e_phentsize
	w.u16(numPhdrs)                  // e_phnum
	w.u16(0)                         // e_shentsize
	w.u16(0)                         // e_shnum
	w.u16(0)                         // e_shstrndx

	// Phdr[0]: PT_LOAD RX — text segment (header + code)
	w.u32(ptLoad)                    // p_type
	w.u32(pfR | pfX)                 // p_flags
	w.u64(0)                         // p_offset
	w.u64(baseAddr)                  // p_vaddr
	w.u64(baseAddr)                  // p_paddr
	w.u64(textFileSize)              // p_filesz
	w.u64(textVMSize)                // p_memsz
	w.u64(elfPageSize)               // p_align

	// Phdr[1]: PT_LOAD RW — BSS segment
	w.u32(ptLoad)                    // p_type
	w.u32(pfR | pfW)                 // p_flags
	w.u64(textVMSize)                // p_offset (past text in file, but filesz=0)
	w.u64(bssVAddr)                  // p_vaddr
	w.u64(bssVAddr)                  // p_paddr
	w.u64(0)                         // p_filesz (no file backing)
	w.u64(bssVMSize)                 // p_memsz (zero-filled by OS)
	w.u64(elfPageSize)               // p_align

	if w.pos() != headerTotal {
		log.Fatalf("ELF header+phdrs = %d bytes, expected %d", w.pos(), headerTotal)
	}

	// Write code
	w.bytes(gen.code)

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
		fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s (%d bytes of x86-64 code)\n", len(w.buf), outpath, codeSize)
	}
}
