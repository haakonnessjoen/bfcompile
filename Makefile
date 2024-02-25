# Eller noe :)

.PRECIOUS: %.s
%.s: %.ssa
	qbe -o $@ $<

.PRECIOUS: %.ssa
%.ssa: %.bf bfcompile
	./bfcompile $< > $@

bfcompile: main.go lexer.go
	go build -o bfcompile *.go

hello: hello.s
	cc -g -o hello hello.s

clean:
	rm -f *.s *.ssa bfcompile hello

distclean: clean
	rm -rf *.dSYM

all: hello