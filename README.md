# Brainfuck compiler

This project parses brainfuck files, optimizes it, and generates [QBE IL](https://c9x.me/compile/doc/il.html) or C code output, ready to be compiled by the LLVM compiler.

The idea came when I saw the Youtuber [tsoding](https://www.youtube.com/watch?v=JTjNoejn4iA) try out QBE, and the language looked very simple, so I thought it should be simple to use it to compile brainfuck.

## Features

Optimizes the token tree (or array really) before it generates QBE IL/C code, by compacting multiple "+" operations to a single operation using add, etc.

## Prerequisites

Unless you are using the C output feature, you need **qbe**, on mac you can just do `brew install qbe`.

## Cross compilation of brainfuck code on mac

On a mac you can cross compile to arm64 and/or amd64 binaries doing the following:

### Specify AMD64 output

```bash
./bfcompile -c -o brainfuck/tictactoe.bf > brainfuck/tictactoe.ssa
qbe -t amd64_apple -o brainfuck/tictactoe.s brainfuck/tictactoe.ssa
cc brainfuck/tictactoe.s -target x86_64-apple-darwin-macho -o tictactoe_amd64
```

### Specify ARM64 output

```bash
./bfcompile -c -o brainfuck/tictactoe.bf > brainfuck/tictactoe.ssa
qbe -t arm64_apple -o brainfuck/tictactoe.s brainfuck/tictactoe.ssa
cc brainfuck/tictactoe.s -target arm64-apple-darwin-macho -o tictactoe_arm64
```
