package generators

import (
	u "bcomp/bfutils"

	"fmt"
	"regexp"
	"strconv"
)

type LoopEntry int
type LLVMGenerator struct {
	refNum int
	debugInfo []string
	debugMap map[string]int
	refStack []string
	loopStack []LoopEntry
	nextJmp int
	vno int
	jumpMap map[string]int
	currentScope string
	currentScopeNum int
	currentLine string
	wordSize int
	f *GeneratorOutput
}

func NewGeneratorHelper(f *GeneratorOutput, wordSize int) *LLVMGenerator {
	return &LLVMGenerator{
		refNum: 0,
		debugInfo: make([]string, 0, 100),
		debugMap: make(map[string]int),
		refStack: make([]string, 1, 100),
		loopStack: make([]LoopEntry, 0, 100),
		currentScope: "",
		currentScopeNum: -1,
		currentLine: "",
		nextJmp: 1,
		vno: 1,
		wordSize: wordSize,
		jumpMap: make(map[string]int),
		f: f,
	}
}

func (g *LLVMGenerator) getJumpLbl(jumptype string, reference int) int {
	lbl := fmt.Sprintf("%s%d", jumptype, reference)
	_, ok := g.jumpMap[lbl]
	if !ok {
		g.jumpMap[lbl] = g.nextJmp
		defer func() {
			g.nextJmp++
		}()
	}
	return g.jumpMap[lbl]
}

func (g *LLVMGenerator) addDebug(name string, format string, args ...interface{}) int {
	if name != "" {
		g.debugMap[name] = g.refNum
	}
	prefix := fmt.Sprintf("!%d = ", g.refNum)
	g.debugInfo = append(g.debugInfo, prefix+fmt.Sprintf(format, args...))
	g.refNum++
	return g.refNum - 1
}

// For forward references
func (g *LLVMGenerator) debugRefPh(name string) string {
	return fmt.Sprintf("$${REPLACE_%s}", name)
}

func (g *LLVMGenerator) debugRef(name string) (ref int) {
	var ok bool
	ref, ok = g.debugMap[name]
	if !ok {
		ref = -1
	}
	return ref
}

func (g *LLVMGenerator) pushRef(ref string) {
	g.refStack = append(g.refStack, ref)
	g.currentScope = ref
	g.currentScopeNum = g.debugRef(ref)
}

func (g *LLVMGenerator) popRef() string {
	if len(g.refStack) > 1 {
		g.refStack = g.refStack[:len(g.refStack)-1]
	}
	g.currentScope = g.refStack[len(g.refStack)-1]
	g.currentScopeNum = g.debugRef(g.currentScope)
	return g.currentScope
}

func (g *LLVMGenerator) addLine(line, column int) (ref string) {
	if !DebugSymbols {
		return
	}
	ref = fmt.Sprintf("L%d_%d", line, column)
	g.addDebug(ref, "!DILocation(line: %d, column: %d, scope: !%d)", line, column, g.currentScopeNum)
	return
}

func (g *LLVMGenerator) addBlock(line, column int) (ref string) {
	if !DebugSymbols {
		return
	}
	ref = fmt.Sprintf("B%d_%d", line, column)
	g.addDebug(ref, "distinct !DILexicalBlock(scope: !%d, file: !%s, line: %d, column: %d)",
		g.currentScopeNum, g.debugRefPh("bf_file"), line, column,
	)
	g.pushRef(ref)
	return
}

func (g *LLVMGenerator) endBlock() {
	if !DebugSymbols {
		return
	}
	g.popRef()
}

func (g *LLVMGenerator) nextv() int {
	g.vno++
	return g.vno
}

func (g *LLVMGenerator) printf(format string, args ...interface{}) {
	g.f.Printf(format, args...)
	if DebugSymbols {
		g.f.Printf(", !dbg !%d\n", g.debugRef(g.currentLine))
	} else {
		g.f.Println("")
	}
}

func (g *LLVMGenerator) OutputDebugInfo() {
	// Ouput debug information with the correct references
	re := regexp.MustCompile(`\$\${REPLACE_([^}]+)}`)
	for _, v := range g.debugInfo {
		newvalue := u.ReplaceAllStringSubmatchFunc(re, v, func(groups []string) string {
			val := g.debugRef(groups[1])
			if val == -1 {
				return "[N/A]"
			}
			return strconv.Itoa(val)
		})
		g.f.Println(newvalue)
	}
}

func (g *LLVMGenerator) printLoadPtr(p1 int) {
	g.printf("  %%p.%d = load  ptr, ptr %%p, align 8", p1)
}

func (g *LLVMGenerator) printLoadValue(v1 int, p1 int) {
	g.printf("  %%v.%d = load i%d,  ptr %%p.%d, align 1", v1, g.wordSize, p1)
}

func (g *LLVMGenerator) printTruncValue(v2, v1 int) int {
	if g.wordSize == 32 {
		return v1
	}
	g.printf("  %%v.%d = trunc i32 %%v.%d to i%d", v2, v1, g.wordSize)
	return v2
}

func (g *LLVMGenerator) printExtendValue(v2, v1 int) int {
	if g.wordSize == 32 {
		return v1
	}
	g.printf("  %%v.%d = zext i%d %%v.%d to i32", v2, g.wordSize, v1)
	return v2
}

func (g *LLVMGenerator) printStoreValue(v2, p1 int) {
	g.printf("  store i%d %%v.%d, ptr %%p.%d, align 1", g.wordSize, v2, p1)
}