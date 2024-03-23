# Brainfuck compiler

This project parses brainfuck files, optimizes it, and generates [QBE IL](https://c9x.me/compile/doc/il.html), C code, Javascript or Brainfuck output. If you output QBE IL, you can use the `qbe` tool to generate assembly that can be compiled with llvm, se examples below.

The idea came when I saw the Youtuber [tsoding](https://www.youtube.com/watch?v=JTjNoejn4iA) try out QBE, and the language looked very simple, so I thought it should be simple to use it to compile brainfuck.

## Features

If you enable optimization, it optimizes the token tree (or array really) before it generates code with the following optimalizations, output is explained with C for simplicity:

- Memory size (default 30KB) and cell size (32,16 or 8 bit) is configurable

- Multiple equal operations are aggregated, for example `++++` would generate `*p += 4` in C

- Opposing operations cancel eachother out. For example `+++--` would generate `*p++` in C

- If opposing operations "overflow" they swap operation, so `++---` would change the initial + operation that is being handeled into a - operation, so the resulting C in this case would be `*p--`

- Optimalization is re-run until no more optimization is possible, so that for example `++-->+->+->+-` would be optimized down to just `*p += 3`

- Second level optimalization; convert simple loops that only do maths, to multiplication instructions. For example; `++[>+++>++<<-]` would optimize to the following C code: `p[0] = 2; p[1] += *p * 3; p[2] += *p * 2;`. It also handles when the iterator is more than 1; for example: `++[>>+++>++<<--]` would generate `p[0] = 2; p[0] =/ 2; p[2] += *p * 3; p[3] += *p * 2;`, so that the result would be half the last example. This optimization cannot be used with Brainfuck output of course.

- Added an `if` before the multiplication operations, in case the operation should not be done at all.

- Just after a loop, we know that *p is 0, so if there is a ADD/SUB operation just after that, we can set it directly. For example `[-]++` would translate to `*p = 2`instead of`*p = 0; *p += 2;`

- For C output, there are a few extra optimalizations, one where `[>]` will use stdlib call memchr() which on some standard libraries are optimized to check 4 bytes at a time. And a second one, that converts `[.>]` to `p += fputs(p, stdout)` as it will both increase the pointer, and let the standard library ouput the string the most optimal way. These optimizations are probably not very noticable in most brainfuck programs though.

- If a loop variable is cleared before the loop ends, convert it to a if statement instead. (This will also handle loops that contains ex-loops that has been converted to multiplications)

Fun fact: It can also output Brainfuck, so you can use it to optimize your brainfuck (only level 1). For example output from this ["C" to bf compiler](https://github.com/elikaski/BF-it) can often be optimized quite a bit, as it does a lot of operations that would cancel eachother out.

## Prerequisites

If you are using qbe output format you need **qbe**. On mac you can just do `brew install qbe`.

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
