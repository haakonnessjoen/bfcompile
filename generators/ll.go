package generators

import (
	u "bcomp/bfutils"
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
)

var DebugSymbols = false

// PrintIR prints the tokens as LLVM Intermediate Representation
func PrintIR(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
	g := NewGeneratorHelper(f, wordSize)

	filename := u.Globals.Get("INPUT_FILENAME")

	if u.Globals.Get("LLVM_DEBUG") == "true" {
		DebugSymbols = true
	}

	f.Printf("; ModuleID = '%s'\n", filename)
	f.Printf("source_filename = \"%s\"\n\n", filename)

	g.pushRef("main")

	if DebugSymbols {
		g.addDebug("l0", "!DIGlobalVariableExpression(var: !%s, expr: !DIExpression())", g.debugRefPh("@mem"))
		g.addDebug("@mem", "distinct !DIGlobalVariable(name: \"mem\", scope: !%s, file: !%s, line: 0, type: !%s, isLocal: false, isDefinition: true)",
			g.debugRefPh("scope"), g.debugRefPh("bf_file"), g.debugRefPh("memtype"),
		)
		g.addDebug("scope", "distinct !DICompileUnit(language: DW_LANG_C, file: !%s, producer: \"%s %s\", isOptimized: %v, runtimeVersion: 0, emissionKind: FullDebug, globals: !%s, splitDebugInlining: false, nameTableKind: None)",
			g.debugRefPh("bf_file"), u.Globals.Get("PACKAGE_NAME"), u.Globals.Get("PACKAGE_VERSION"), false, g.debugRefPh("globals"),
		)
		g.addDebug("bf_file", "!DIFile(filename: \"%s\", directory: \"%s\")",
			filename, os.Getenv("PWD"),
		)
		g.addDebug("globals", "!{!%s}", g.debugRefPh("l0"))
		g.addDebug("memtype", "!DICompositeType(tag: DW_TAG_array_type, baseType: !%s, size: %d, elements: !%s)", g.debugRefPh("uinttype"), memorySize*wordSize, g.debugRefPh("elements"))
		g.addDebug("uinttype", "!DIDerivedType(tag: DW_TAG_typedef, name: \"uint%d\", file: !%d, line: 1, baseType: !%s)", wordSize, g.debugRef("bf_file"), g.debugRefPh("baseuinttype"))
		g.addDebug("baseuinttype", "!DIBasicType(name: \"unsigned char\", size: %d, encoding: DW_ATE_unsigned_char)", wordSize)
		g.addDebug("elements", "!{!%s}", g.debugRefPh("elementscount"))
		g.addDebug("elementscount", "!DISubrange(count: %d)", memorySize)
	}

	g.addDebug("flag1", "!{i32 7, !\"Dwarf Version\", i32 4}")
	g.addDebug("flag2", "!{i32 2, !\"Debug Info Version\", i32 3}")
	g.addDebug("flag3", "!{i32 1, !\"wchar_size\", i32 4}")
	g.addDebug("flag4", "!{i32 8, !\"PIC Level\", i32 2}")
	g.addDebug("flag5", "!{i32 7, !\"uwtable\", i32 1}")
	g.addDebug("flag6", "!{i32 7, !\"frame-pointer\", i32 1}")
	g.addDebug("ident", "!{!\"%s %s\"}", u.Globals.Get("PACKAGE_NAME"), u.Globals.Get("PACKAGE_VERSION"))

	if DebugSymbols {
		g.addDebug("main", "distinct !DISubprogram(name: \"main\", scope: !%s, file: !%s, line: 1, type: !%s, scopeLine: 1, spFlags: DISPFlagDefinition, unit: !%s, retainedNodes: !%s)",
			g.debugRefPh("bf_file"), g.debugRefPh("bf_file"), g.debugRefPh("int32type"), g.debugRefPh("scope"), g.debugRefPh("retainedNodes"),
		)

		if g.currentScope == "main" {
			g.currentScopeNum = g.debugRef("main")
		}
		g.addDebug("int32type", "!DISubroutineType(types: !%s)", g.debugRefPh("int32typeref"))
		g.addDebug("int32typeref", "!{!%s}", g.debugRefPh("int32typedef"))
		g.addDebug("int32typedef", "!DIBasicType(name: \"int\", size: 32, encoding: DW_ATE_signed)")
		g.addDebug("retainedNodes", "!{}")

		g.addDebug("pvar", "!DILocalVariable(name: \"p\", scope: !%s, file: !%s, line: 1, type: !%s)", g.debugRefPh("main"), g.debugRefPh("bf_file"), g.debugRefPh("mempointertype"))
		g.addDebug("mempointertype", "!DIDerivedType(tag: DW_TAG_pointer_type, baseType: !%s, size: 64)", g.debugRefPh("uinttype"))
	}

	if DebugSymbols {
		f.Printf("@mem = common global [%d x i%d] zeroinitializer, align 1, !dbg !%d\n\n", memorySize, wordSize, 0)
		f.Printf("define i32 @main() #0 !dbg !%d {\n", g.debugRef("main"))
	} else {
		f.Printf("@mem = common global [%d x i%d] zeroinitializer, align 1\n\n", memorySize, wordSize)
		f.Println("define i32 @main() #0 {")
	}

	f.Println("  %p = alloca ptr, align 8")
	if DebugSymbols {
		f.Printf("  call void @llvm.dbg.declare(metadata ptr %%p, metadata !%d, metadata !DIExpression()), !dbg !%d\n", g.debugRef("pvar"), g.refNum+1)
		f.Printf("  store ptr @mem, ptr %%p, align 8, !dbg !%d\n", g.refNum+1)
	} else {
		f.Printf("  store ptr @mem, ptr %%p, align 8\n")
	}

	for _, t := range tokens {
		if includeComments {
			f.Printf("; Pos %d:%d %s (%s, %d, %d)\n", t.Pos.Line, t.Pos.Column, t.Tok.Character, t.Tok.TokenName, t.Extra, t.Extra2)
		}

		switch t.Tok.Tok {
		case l.ADD:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			v1 := g.nextv()
			p1 := g.nextv()
			v2 := g.nextv()

			g.printLoadPtr(p1)
			g.printLoadValue(v1, p1)

			if t.Extra != 1 {
				oldv1 := v1
				v1 = g.nextv()
				v1 = g.printExtendValue(v1, oldv1)
				g.printf("  %%v.%d = add nsw i32 %%v.%d, %d", v2, v1, t.Extra)
				v3 := g.nextv()
				v3 = g.printTruncValue(v3, v2)
				g.printStoreValue(v3, p1)
			} else {
				g.printf("  %%v.%d = add nsw i%d %%v.%d, %d", v2, wordSize, v1, t.Extra)
				g.printStoreValue(v2, p1)
			}
		case l.SUB:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			v1 := g.nextv()
			p1 := g.nextv()
			v2 := g.nextv()

			g.printLoadPtr(p1)
			g.printLoadValue(v1, p1)

			if t.Extra != 1 {
				oldv1 := v1
				v1 = g.nextv()
				v1 = g.printExtendValue(v1, oldv1)
				g.printf("  %%v.%d = add nsw i32 %%v.%d, %d", v2, v1, -t.Extra)
				v3 := g.nextv()
				v3 = g.printTruncValue(v3, v2)
				g.printStoreValue(v3, p1)
			} else {
				g.printf("  %%v.%d = add nsw i%d %%v.%d, %d", v2, wordSize, v1, -t.Extra)
				g.printStoreValue(v2, p1)
			}
		case l.INCP:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			p1 := g.nextv()
			p2 := g.nextv()
			g.printLoadPtr(p1)
			g.printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i32 %d", p2, wordSize, p1, t.Extra)
			g.printf("  store ptr %%p.%d, ptr %%p, align 8", p2)
		case l.DECP:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			p1 := g.nextv()
			p2 := g.nextv()
			g.printLoadPtr(p1)
			g.printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i32 %d", p2, wordSize, p1, -t.Extra)
			g.printf("  store ptr %%p.%d, ptr %%p, align 8", p2)
		case l.OUT:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			for i := 0; i < t.Extra; i++ {
				v1 := g.nextv()
				p1 := g.nextv()
				v2 := g.nextv()
				g.printLoadPtr(p1)
				g.printLoadValue(v1, p1)
				v2 = g.printExtendValue(v2, v1)
				g.printf("  %%v.%d = call i32 @putchar(i32 noundef %%v.%d)", g.nextv(), v2)
			}
		case l.IN:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			var v1 int
			for i := 0; i < t.Extra; i++ {
				v1 = g.nextv()
				g.printf("  %%v.%d = call i32 @getchar()", v1)
			}
			v2 := g.nextv()
			v2 = g.printTruncValue(v2, v1)
			p1 := g.nextv()
			g.printLoadPtr(p1)
			g.printStoreValue(v2, p1)
		case l.JMPF:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			g.addBlock(t.Pos.Line, t.Pos.Column)
			jmplabel := g.getJumpLbl("f", t.Extra)
			g.printf("  br label %%j%d", jmplabel)
			f.Printf("\nj%d:\n", jmplabel)

			p1 := g.nextv()
			v1 := g.nextv()
			v2 := g.nextv()
			g.printLoadPtr(p1)
			g.printLoadValue(v1, p1)
			g.printf("  %%v.%d = icmp ne i%d %%v.%d, 0", v2, wordSize, v1)

			fdlabel := g.getJumpLbl("fd", t.Extra)
			bdlabel := g.getJumpLbl("bd", t.Extra)
			g.printf("  br i1 %%v.%d, label %%j%d, label %%j%d", v2, fdlabel, bdlabel)
			f.Printf("\nj%d:\n", fdlabel)
			g.loopStack = append(g.loopStack, LoopEntry(g.debugRef(g.currentLine)))
		case l.JMPB:
			g.endBlock()

			jumpend := fmt.Sprintf("JMPB%d", t.Extra)
			if DebugSymbols {
				g.addDebug(jumpend, "distinct !{!%d, !%d, !%d, !%s}",
					g.refNum, g.loopStack[len(g.loopStack)-1], g.debugRef(g.currentLine)+2, g.debugRefPh("mustProcessRef"),
				)
			} else {
				g.addDebug(jumpend, "distinct !{!%d, !%s}", g.refNum, g.debugRefPh("mustProcessRef"))
			}

			g.loopStack = g.loopStack[:len(g.loopStack)-1]
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)

			g.printf(" br label %%j%d, !llvm.loop !%d", g.getJumpLbl("f", t.Extra), g.debugRef(jumpend))
			f.Printf("\nj%d:\n", g.getJumpLbl("bd", t.Extra))

			if g.debugRef("mustProcessRef") == -1 {
				g.addDebug("mustProcessRef", "!{!\"llvm.loop.mustprogress\"}")
			}
		case l.MUL:
			// p[%d] += *p * %d;
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)

			multiplier := t.Extra
			ptr := t.Extra2

			p1 := g.nextv()
			v1 := g.nextv()
			v2 := g.nextv()
			v3 := g.nextv()
			// v3 = *p * %d
			g.printLoadPtr(p1)
			g.printLoadValue(v1, p1)
			v2 = g.printExtendValue(v2, v1)
			g.printf("  %%v.%d = mul nsw i32 %%v.%d, %d", v3, v2, multiplier)

			p2 := g.nextv()
			p3 := g.nextv()
			v4 := g.nextv()
			v5 := g.nextv()
			// p3 = p[%d]
			// v5 = *(p[%d])
			g.printLoadPtr(p2)
			g.printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p3, wordSize, p2, ptr)
			g.printLoadValue(v4, p3)
			v5 = g.printExtendValue(v5, v4)

			v6 := g.nextv()
			v7 := g.nextv()
			// v7 = trunc(v5 + v3)
			g.printf("  %%v.%d = add nsw i32 %%v.%d, %%v.%d", v6, v5, v3)
			v7 = g.printTruncValue(v7, v6)
			// *p3 = v7
			g.printStoreValue(v7, p3)

		case l.DIV:
			// p[%d] /= %d;
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			ptr := t.Extra2

			p1 := g.nextv()
			// fetch *p or p[%d] to p1
			g.printLoadPtr(p1)
			if ptr != 0 {
				oldp1 := p1
				p1 = g.nextv()
				g.printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p1, wordSize, oldp1, ptr)
			}

			v1 := g.nextv()
			v2 := g.nextv()
			v3 := g.nextv()
			v4 := g.nextv()
			// int32_t v1 = *p
			g.printLoadValue(v1, p1)
			v2 = g.printExtendValue(v2, v1)
			// int32_t v3 = v2 / t.Extra
			g.printf("  %%v.%d = sdiv i32 %%v.%d, %d", v3, v2, t.Extra)
			// *p = trunc(v3)
			v4 = g.printTruncValue(v4, v3)
			g.printStoreValue(v4, p1)

		case l.BZ:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			p1 := g.nextv()
			v1 := g.nextv()
			v2 := g.nextv()
			// v1 = *p
			g.printLoadPtr(p1)
			g.printLoadValue(v1, p1)
			// v2 = v1 != 0
			g.printf("  %%v.%d = icmp ne i%d %%v.%d, 0", v2, wordSize, v1)
			g.printf("  br i1 %%v.%d, label %%j%d, label %%j%d", v2, g.getJumpLbl("f", t.Extra), g.getJumpLbl("if", t.Extra))

			f.Printf("\nj%d:\n", g.getJumpLbl("f", t.Extra))
		case l.LBL:
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			g.printf("  br label %%j%d", g.getJumpLbl("if", t.Extra))
			f.Printf("\nj%d:\n", g.getJumpLbl("if", t.Extra))
		case l.MOV:
			// p[%d] = %d;
			g.currentLine = g.addLine(t.Pos.Line, t.Pos.Column)
			ptr := t.Extra2
			value := t.Extra

			p1 := g.nextv()
			g.printLoadPtr(p1)

			if ptr != 0 {
				pold := p1
				p1 = g.nextv()
				g.printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p1, wordSize, pold, ptr)
			}

			g.printf("  store i%d %d, ptr %%p.%d, align 1", wordSize, value, p1)
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	g.printf("  ret i32 0")
	f.Println("}\n")

	declarationCounter := 1

	if DebugSymbols {
		f.Printf("declare void @llvm.dbg.declare(metadata, metadata, metadata) #%d\n", declarationCounter)
		declarationCounter++
	}
	f.Printf("declare i32 @getchar() #%d\n", declarationCounter)
	declarationCounter++
	f.Printf("declare i32 @putchar(i32 noundef) #%d\n\n", declarationCounter)
	declarationCounter++

	f.Println("attributes #0 = { nounwind norecurse ssp uwtable  \"frame-pointer\"=\"all\" \"min-legal-vector-width\"=\"0\" \"no-trapping-math\"=\"true\" \"probe-stack\"=\"___chkstk_darwin\" \"stack-protector-buffer-size\"=\"8\" \"tune-cpu\"=\"generic\" }")
	f.Println("attributes #1 = { nocallback nofree nosync nounwind readnone speculatable willreturn }")
	f.Println("attributes #2 = { \"darwin-stkchk-strong-link\" \"frame-pointer\"=\"all\" \"no-trapping-math\"=\"true\" \"probe-stack\"=\"___chkstk_darwin\" \"stack-protector-buffer-size\"=\"8\" \"tune-cpu\"=\"generic\" }\n")

	f.Printf("!llvm.module.flags = !{!%d, !%d, !%d, !%d, !%d, !%d}\n",
		g.debugRef("flag1"), g.debugRef("flag2"), g.debugRef("flag3"), g.debugRef("flag4"), g.debugRef("flag5"), g.debugRef("flag6"),
	)
	if DebugSymbols {
		f.Printf("!llvm.dbg.cu = !{!%d}\n", g.debugRef("scope"))
	}
	f.Printf("!llvm.ident = !{!%d}\n\n", g.debugRef("ident"))

	g.OutputDebugInfo()
}
