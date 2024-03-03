# Eller noe :)

SRC_DIR := brainfuck
SRCS := $(wildcard $(SRC_DIR)/*.bf)
EXECS := $(patsubst $(SRC_DIR)/%.bf,%,$(SRCS))

.PRECIOUS: %.s
%.s: %.ssa
	qbe -o $@ $<

.PRECIOUS: %.ssa
%.ssa: %.bf bfcompile
	./bfcompile -o -g qbe $< > $@

.DEFAULT_GOAL := all

GOSOURCES := $(wildcard *.go) $(wildcard generators/*.go) $(wildcard lexer/*.go) $(wildcard parser/*.go) $(wildcard interpreter/*.go)
bfcompile: $(GOSOURCES)
	go build -o bfcompile $(wildcard *.go)

$(EXECS): % : $(SRC_DIR)/%.s
	cc -Os $< -o $@

clean:
	rm -f bfcompile $(SRC_DIR)/*.s $(SRC_DIR)/*.ssa $(EXECS)
	rm -rf *.dSYM

all: $(EXECS)