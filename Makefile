# Eller noe :)

.PRECIOUS: %.s
%.s: %.ssa
	qbe -o $@ $<

.PRECIOUS: %.ssa
%.ssa: %.bf bfcompile
	./bfcompile $< > $@

bfcompile: main.go lexer.go
	go build -o bfcompile *.go

hello: brainfuck/hello.s
	cc -g -o hello brainfuck/hello.s

clean:
	rm -f brainfuck/*.s brainfuck/*.ssa bfcompile hello
	rm -rf *.dSYM

all: hello