package generators

import (
	u "bcomp/bfutils"
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
)

type LoopEntry int

var DebugSymbols = false

// PrintIR prints the tokens as LLVM Intermediate Representation
func PrintIR(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
	refNum := 0
	debugInfo := make([]string, 0, 100)
	debugMap := make(map[string]int)
	refStack := make([]string, 1, 100)
	refStack[0] = "main"
	loopStack := make([]LoopEntry, 0, 100)
	currentScope := ""
	currentScopeNum := -1
	nextJmp := 1
	jumpMap := make(map[string]int)

	filename := u.Globals.Get("INPUT_FILENAME")

	if u.Globals.Get("LLVM_DEBUG") == "true" {
		DebugSymbols = true
	}

	getJumpLbl := func(jumptype string, reference int) int {
		lbl := fmt.Sprintf("%s%d", jumptype, reference)
		_, ok := jumpMap[lbl]
		if !ok {
			jumpMap[lbl] = nextJmp
			defer func() {
				nextJmp++
			}()
		}
		return jumpMap[lbl]
	}

	addDebug := func(name string, format string, args ...interface{}) int {
		if name != "" {
			debugMap[name] = refNum
		}
		prefix := fmt.Sprintf("!%d = ", refNum)
		debugInfo = append(debugInfo, prefix+fmt.Sprintf(format, args...))
		refNum++
		return refNum - 1
	}

	// For forward references
	debugRefPh := func(name string) string {
		return fmt.Sprintf("$${REPLACE_%s}", name)
	}

	debugRef := func(name string) (ref int) {
		var ok bool
		ref, ok = debugMap[name]
		if !ok {
			ref = -1
		}
		return ref
	}

	pushRef := func(ref string) {
		refStack = append(refStack, ref)
		currentScope = ref
		currentScopeNum = debugRef(ref)
	}

	popRef := func() string {
		if len(refStack) > 1 {
			refStack = refStack[:len(refStack)-1]
		}
		currentScope = refStack[len(refStack)-1]
		currentScopeNum = debugRef(currentScope)
		return currentScope
	}

	addLine := func(line, column int) (ref string) {
		if !DebugSymbols {
			return
		}
		ref = fmt.Sprintf("L%d_%d", line, column)
		addDebug(ref, "!DILocation(line: %d, column: %d, scope: !%d)", line, column, currentScopeNum)
		return
	}

	addBlock := func(line, column int) (ref string) {
		if !DebugSymbols {
			return
		}
		ref = fmt.Sprintf("B%d_%d", line, column)
		addDebug(ref, "distinct !DILexicalBlock(scope: !%d, file: !%s, line: %d, column: %d)",
			currentScopeNum, debugRefPh("bf_file"), line, column,
		)
		pushRef(ref)
		return
	}

	endBlock := func() {
		if !DebugSymbols {
			return
		}
		popRef()
	}

	f.Printf("; ModuleID = '%s'\n", filename)
	f.Printf("source_filename = \"%s\"\n\n", filename)

	pushRef("main")

	if DebugSymbols {
		addDebug("l0", "!DIGlobalVariableExpression(var: !%s, expr: !DIExpression())", debugRefPh("@mem"))
		addDebug("@mem", "distinct !DIGlobalVariable(name: \"mem\", scope: !%s, file: !%s, line: 0, type: !%s, isLocal: false, isDefinition: true)",
			debugRefPh("scope"), debugRefPh("bf_file"), debugRefPh("memtype"),
		)
		addDebug("scope", "distinct !DICompileUnit(language: DW_LANG_C, file: !%s, producer: \"%s %s\", isOptimized: %v, runtimeVersion: 0, emissionKind: FullDebug, globals: !%s, splitDebugInlining: false, nameTableKind: None)",
			debugRefPh("bf_file"), u.Globals.Get("PACKAGE_NAME"), u.Globals.Get("PACKAGE_VERSION"), false, debugRefPh("globals"),
		)
		addDebug("bf_file", "!DIFile(filename: \"%s\", directory: \"%s\")",
			filename, os.Getenv("PWD"),
		)
		addDebug("globals", "!{!%s}", debugRefPh("l0"))
		addDebug("memtype", "!DICompositeType(tag: DW_TAG_array_type, baseType: !%s, size: %d, elements: !%s)", debugRefPh("uinttype"), memorySize*wordSize, debugRefPh("elements"))
		addDebug("uinttype", "!DIDerivedType(tag: DW_TAG_typedef, name: \"uint%d\", file: !%d, line: 1, baseType: !%s)", wordSize, debugRef("bf_file"), debugRefPh("baseuinttype"))
		addDebug("baseuinttype", "!DIBasicType(name: \"unsigned char\", size: %d, encoding: DW_ATE_unsigned_char)", wordSize)
		addDebug("elements", "!{!%s}", debugRefPh("elementscount"))
		addDebug("elementscount", "!DISubrange(count: %d)", memorySize)
	}

	addDebug("flag1", "!{i32 7, !\"Dwarf Version\", i32 4}")
	addDebug("flag2", "!{i32 2, !\"Debug Info Version\", i32 3}")
	addDebug("flag3", "!{i32 1, !\"wchar_size\", i32 4}")
	addDebug("flag4", "!{i32 8, !\"PIC Level\", i32 2}")
	addDebug("flag5", "!{i32 7, !\"uwtable\", i32 1}")
	addDebug("flag6", "!{i32 7, !\"frame-pointer\", i32 1}")
	addDebug("ident", "!{!\"%s %s\"}", u.Globals.Get("PACKAGE_NAME"), u.Globals.Get("PACKAGE_VERSION"))

	if DebugSymbols {
		addDebug("main", "distinct !DISubprogram(name: \"main\", scope: !%s, file: !%s, line: 1, type: !%s, scopeLine: 1, spFlags: DISPFlagDefinition, unit: !%s, retainedNodes: !%s)",
			debugRefPh("bf_file"), debugRefPh("bf_file"), debugRefPh("int32type"), debugRefPh("scope"), debugRefPh("retainedNodes"),
		)

		if currentScope == "main" {
			currentScopeNum = debugRef("main")
		}
		addDebug("int32type", "!DISubroutineType(types: !%s)", debugRefPh("int32typeref"))
		addDebug("int32typeref", "!{!%s}", debugRefPh("int32typedef"))
		addDebug("int32typedef", "!DIBasicType(name: \"int\", size: 32, encoding: DW_ATE_signed)")
		addDebug("retainedNodes", "!{}")

		addDebug("pvar", "!DILocalVariable(name: \"p\", scope: !%s, file: !%s, line: 1, type: !%s)", debugRefPh("main"), debugRefPh("bf_file"), debugRefPh("mempointertype"))
		addDebug("mempointertype", "!DIDerivedType(tag: DW_TAG_pointer_type, baseType: !%s, size: 64)", debugRefPh("uinttype"))
	}

	if DebugSymbols {
		f.Printf("@mem = common global [%d x i%d] zeroinitializer, align 1, !dbg !%d\n\n", memorySize, wordSize, 0)
		f.Printf("define i32 @main() #0 !dbg !%d {\n", debugRef("main"))
	} else {
		f.Printf("@mem = common global [%d x i%d] zeroinitializer, align 1\n\n", memorySize, wordSize)
		f.Println("define i32 @main() #0 {")
	}

	f.Println("  %v = alloca i8, align 1")
	f.Println("  %p = alloca ptr, align 8")
	f.Println("  store i32 0, ptr %v, align 1")
	if DebugSymbols {
		f.Printf("  call void @llvm.dbg.declare(metadata ptr %%p, metadata !%d, metadata !DIExpression()), !dbg !%d\n", debugRef("pvar"), refNum+1)
		f.Printf("  store ptr @mem, ptr %%p, align 8, !dbg !%d\n", refNum+1)
	} else {
		f.Printf("  store ptr @mem, ptr %%p, align 8\n")
	}
    f.Printf("  %%v.val = load i32, ptr %%v, align 4")

	var currentLine string
	var vNo int = 0

	nextv := func() int {
		vNo++
		return vNo
	}

	printf := func(format string, args ...interface{}) {
		f.Printf(format, args...)
		if DebugSymbols {
			f.Printf(", !dbg !%d\n", debugRef(currentLine))
		} else {
			f.Println("")
		}
	}

	for _, t := range tokens {
		if includeComments {
			f.Printf("; Pos %d:%d %s (%s, %d, %d)\n", t.Pos.Line, t.Pos.Column, t.Tok.Character, t.Tok.TokenName, t.Extra, t.Extra2)
		}

		switch t.Tok.Tok {
		case l.ADD:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			v1 := nextv()
			p1 := nextv()
			v2 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)

			if t.Extra != 1 {
				oldv1 := v1
				v1 = nextv()
				printf("  %%v.%d = zext i%d %%v.%d to i32", v1, wordSize, oldv1)
				printf("  %%v.%d = add nsw i32 %%v.%d, %d", v2, v1, t.Extra)
			} else {
				printf("  %%v.%d = add nsw i%d %%v.%d, %d", v2, wordSize, v1, t.Extra)
			}


			if t.Extra != 1 {
				v3 := nextv()
				printf("  %%v.%d = trunc i32 %%v.%d to i%d", v3, v2, wordSize)
				printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v3, p1)
			} else {
				printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v2, p1)
			}
		case l.SUB:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			v1 := nextv()
			p1 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
			v2 := nextv()
			printf("  %%v.%d = add i%d %%v.%d, %d", v2, wordSize, v1, -t.Extra)
			printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v2, p1)
		case l.INCP:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			p1 := nextv()
			p2 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i32 %d", p2, wordSize, p1, t.Extra)
			printf("  store ptr %%p.%d, ptr %%p, align 8", p2)
		case l.DECP:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			p1 := nextv()
			p2 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i32 %d", p2, wordSize, p1, -t.Extra)
			printf("  store ptr %%p.%d, ptr %%p, align 8", p2)
		case l.OUT:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			for i := 0; i < t.Extra; i++ {
				v1 := nextv()
				p1 := nextv()
				printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
				printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
				v2 := nextv()
				printf("  %%v.%d = zext i%d %%v.%d to i32", v2, wordSize, v1)
				printf("  %%v.%d = call i32 @putchar(i32 noundef %%v.%d)", nextv(), v2)
			}
		case l.IN:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			var v2 int
			for i := 0; i < t.Extra; i++ {
				v1 := nextv()
				printf("  %%v.%d = call i32 @getchar()", v1)
				v2 = nextv()
				printf("  %%v.%d = trunc i32 %%v.%d to i%d", v2, v1, wordSize)
			}
			p1 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v2, p1)
		case l.JMPF:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			addBlock(t.Pos.Line, t.Pos.Column)
			jmplabel := getJumpLbl("f", t.Extra)
			printf("  br label %%j%d", jmplabel)
			f.Printf("\nj%d:\n", jmplabel)
			p1 := nextv()
			v1 := nextv()
			v2 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
			printf("  %%v.%d = icmp ne i%d %%v.%d, 0", v2, wordSize, v1)
			fdlabel := getJumpLbl("fd", t.Extra)
			bdlabel := getJumpLbl("bd", t.Extra)
			printf("  br i1 %%v.%d, label %%j%d, label %%j%d", v2, fdlabel, bdlabel)
			f.Printf("\nj%d:\n", fdlabel)
			loopStack = append(loopStack, LoopEntry(debugRef(currentLine)))
		case l.JMPB:
			endBlock()

			jumpend := fmt.Sprintf("JMPB%d", t.Extra)
			if DebugSymbols {
				addDebug(jumpend, "distinct !{!%d, !%d, !%d, !%s}",
					refNum, loopStack[len(loopStack)-1], debugRef(currentLine)+2, debugRefPh("mustProcessRef"),
				)
			} else {
				addDebug(jumpend, "distinct !{!%d, !%s}", refNum, debugRefPh("mustProcessRef"))
			}

			loopStack = loopStack[:len(loopStack)-1]
			currentLine = addLine(t.Pos.Line, t.Pos.Column)

			printf(" br label %%j%d, !llvm.loop !%d", getJumpLbl("f", t.Extra), debugRef(jumpend))
			f.Printf("\nj%d:\n", getJumpLbl("bd", t.Extra))

			if debugRef("mustProcessRef") == -1 {
				addDebug("mustProcessRef", "!{!\"llvm.loop.mustprogress\"}")
			}
		case l.MUL:
			// p[%d] += *p * %d;
			currentLine = addLine(t.Pos.Line, t.Pos.Column)

			multiplier := t.Extra
			ptr := t.Extra2

			p1 := nextv()
			v1 := nextv()
			v2 := nextv()
			v3 := nextv()
			// v3 = *p * %d
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
			printf("  %%v.%d = zext i%d %%v.%d to i32", v2, wordSize, v1)
			printf("  %%v.%d = mul nsw i32 %%v.%d, %d", v3, v2, multiplier)

			p2 := nextv()
			p3 := nextv()
			v4 := nextv()
			v5 := nextv()
			// p3 = p[%d]
			// v5 = *(p[%d])
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p2)
			printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p3, wordSize, p2, ptr)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v4, wordSize, p3)
			printf("  %%v.%d = zext i%d %%v.%d to i32", v5, wordSize, v4)

			v6 := nextv()
			v7 := nextv()
			// v7 = trunc(v5 + v3)
			printf("  %%v.%d = add nsw i32 %%v.%d, %%v.%d", v6, v5, v3)
			printf("  %%v.%d = trunc i32 %%v.%d to i%d", v7, v6, wordSize)
			// *p3 = v7
			printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v7, p3)

		case l.DIV:
			// p[%d] /= %d;
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			ptr := t.Extra2

			p1 := nextv()
			// fetch *p or p[%d] to p1
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			if ptr != 0 {
				oldp1 := p1
				p1 = nextv()
				printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p1, wordSize, oldp1, ptr)
			}

			v1 := nextv()
			v2 := nextv()
			v3 := nextv()
			v4 := nextv()
			// int32_t v1 = *p
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
			printf("  %%v.%d = zext i%d %%v.%d to i32", v2, wordSize, v1)
			// int32_t v3 = v2 / t.Extra
			printf("  %%v.%d = sdiv i32 %%v.%d, %d", v3, v2, t.Extra)
			// *p = trunc(v3)
			printf("  %%v.%d = trunc i32 %%v.%d to i%d", v4, v3, wordSize)
			printf("  store i%d %%v.%d, ptr %%p.%d, align 1", wordSize, v4, p1)

		case l.BZ:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			p1 := nextv()
			v1 := nextv()
			v2 := nextv()
			// v1 = *p
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)
			printf("  %%v.%d = load i%d, ptr %%p.%d, align 1", v1, wordSize, p1)
			// v2 = v1 != 0
			printf("  %%v.%d = icmp ne i%d %%v.%d, 0", v2, wordSize, v1)
			printf("  br i1 %%v.%d, label %%j%d, label %%j%d", v2, getJumpLbl("f", t.Extra), getJumpLbl("if", t.Extra))

			f.Printf("\nj%d:\n", getJumpLbl("f", t.Extra))
		case l.LBL:
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			printf("  br label %%j%d", getJumpLbl("if", t.Extra))
			f.Printf("\nj%d:\n", getJumpLbl("if", t.Extra))
		case l.MOV:
			// p[%d] = %d;
			currentLine = addLine(t.Pos.Line, t.Pos.Column)
			ptr := t.Extra2
			value := t.Extra

			p1 := nextv()
			printf("  %%p.%d = load ptr, ptr %%p, align 8", p1)

			if ptr != 0 {
				pold := p1
				p1 = nextv()
				printf("  %%p.%d = getelementptr inbounds i%d, ptr %%p.%d, i64 %d", p1, wordSize, pold, ptr)
			}

			// We should probably use a bigger "wordsize" for pointer references
			printf("  store i%d %d, ptr %%p.%d, align 1", wordSize, value, p1)
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	printf("  ret i32 0")
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

	f.Println("attributes #0 = { noinline nounwind optnone ssp uwtable  \"frame-pointer\"=\"all\" \"min-legal-vector-width\"=\"0\" \"no-trapping-math\"=\"true\" \"probe-stack\"=\"___chkstk_darwin\" \"stack-protector-buffer-size\"=\"8\" \"tune-cpu\"=\"generic\" }")
	f.Println("attributes #1 = { nocallback nofree nosync nounwind readnone speculatable willreturn }")
	f.Println("attributes #2 = { \"darwin-stkchk-strong-link\" \"frame-pointer\"=\"all\" \"no-trapping-math\"=\"true\" \"probe-stack\"=\"___chkstk_darwin\" \"stack-protector-buffer-size\"=\"8\" \"tune-cpu\"=\"generic\" }\n")

	f.Printf("!llvm.module.flags = !{!%d, !%d, !%d, !%d, !%d, !%d}\n",
		debugRef("flag1"), debugRef("flag2"), debugRef("flag3"), debugRef("flag4"), debugRef("flag5"), debugRef("flag6"),
	)
	if DebugSymbols {
		f.Printf("!llvm.dbg.cu = !{!%d}\n", debugRef("scope"))
	}
	f.Printf("!llvm.ident = !{!%d}\n\n", debugRef("ident"))

	// Ouput debug information with the correct references
	re := regexp.MustCompile(`\$\${REPLACE_([^}]+)}`)
	for _, v := range debugInfo {
		newvalue := u.ReplaceAllStringSubmatchFunc(re, v, func(groups []string) string {
			val := debugRef(groups[1])
			if val == -1 {
				return "[N/A]"
			}
			return strconv.Itoa(val)
		})
		f.Println(newvalue)
	}
}
