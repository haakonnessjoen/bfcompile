# Brainfuck compiler

This project parses brainfuck files, optimizes it, and generates debuggable [LLVM IR](https://llvm.org/), [QBE IL](https://c9x.me/compile/doc/il.html), C code, Javascript or Brainfuck output.

If you output LLVM IR and compile your binaries with clang, you can use the `lldb` debugger tool to step through your brainfuck source, while watching the assembly internals! And you can debug the memory with `p mem` or `p p[0]` for example, or find the current pointer location with `p p-mem` and of course output the assembly code with `disassemble`.

![lldb in action](doc/lldb.gif)

The initial idea came when I saw the Youtuber [tsoding](https://www.youtube.com/watch?v=JTjNoejn4iA) try out QBE, and the language looked very simple, so I thought it should be simple to use it to compile brainfuck. And then later I found out that LLVM IR isn't that hard either, to do basic stuff.

## Output formats

* LLVM Intermediate Representation
* QBE Intermediate Language
* C
* Javascript (Node.js flavored)
* Brainfuck

## Optimizations

If you enable optimization, it optimizes the token stream before it generates code with the following optimalizations, output is explained with C for readability:

- Memory size (default 30KB) and cell size (32,16 or 8 bit) is configurable

- Multiple equal operations are aggregated, for example `++++` would generate `*p += 4` in C

- Opposing operations cancel eachother out. For example `+++--` would generate `*p++` in C

- If opposing operations "overflow" they swap operation, so `++---` would change the initial + operation that is being handeled into a - operation, so the resulting C in this case would be `*p--`

- Optimalization is re-run until no more optimization is possible, so that for example `++-->+->+->+-` would be optimized down to just `*p += 3`

- Second level optimalization; convert simple loops that only do maths, to multiplication instructions. For example; `++[>+++>++<<-]` would optimize to the following C code: `p[0] = 2; p[1] += *p * 3; p[2] += *p * 2;`. It also handles when the iterator is higher than 1; for example: `++[>>+++>++<<--]` would generate `p[0] = 2; p[0] =/ 2; p[2] += *p * 3; p[3] += *p * 2;`, so that the result would be half the last example.

- Added an `if` before the multiplication operations, in case the operation should not be done at all.

- Just after a loop, we know that *p is 0, so if there is a ADD/SUB operation just after that, we can set it directly. For example `[-]++` would translate to `*p = 2`instead of`*p = 0; *p += 2;`

- For C output, there are a few extra optimalizations, one where `[>]` will use stdlib call memchr() which on some standard libraries are optimized to check 4 bytes at a time. And a second one, that converts `[.>]` to `p += fputs(p, stdout)` as it will both increase the pointer, and let the standard library ouput the string the most optimal way. These optimizations are probably not very noticable in most brainfuck programs though.

- If a loop variable is cleared before the loop ends, convert it to a if statement instead. (This will also handle loops that contains ex-loops that has been converted to multiplications)

Fun fact: It can also output Brainfuck, so you can use it to optimize your brainfuck (only level 1 optimizations). For example output from this ["C" to bf compiler](https://github.com/elikaski/BF-it) can often be optimized quite a bit, as it does a lot of operations that would cancel eachother out.

## Known limitations

* Some of the second level optimizations expect you not to use negative overflow in multiplication, like this example: `--[------->++<]>.`, this should normally give you 36, but if you enable the optimizer, it will result in 72, because the optimizer starts up trying to divide 254 by 7, which isn't integer divisable. I have tested a lot of brainfuck code written by other people or compilers, and I seldom see this, but I have so far only found this issue with a ["text-to-bf"](https://copy.sh/brainfuck/text.html) script that tries to optimize it's output by utilizing higher numbers by going below 0.

* Instead of learning the depths of LLVM IR, I have used the output of clang to help me on my way, by using defaults found in the output of those files. This means that the output of the LLVM IR generator, is probably highly dependant on compiling for mac, and might not work for other architectures etc, since, even if LLVM IR is architecture agnostic, it has a lot of features you can enable if you know you are outputting to a specific architecture. But maybe I will do more work on this later. At the current time, this project is more of a proof of concept and R&D.

## Prerequisites

*LLVM*: If you are using llvm output, you need llvm/clang toolset. Included with XCode on mac.

*QBE*: If you are using qbe output format you need **qbe** to generate assembly code (and llvm to compile). On mac you can just do `brew install qbe`.

# Compilation with LLVM

To compile your project for your current architecture, you should be able to do:

```bash
bfcompile -o -g llvm -c brainfuck/tictactoe.bf > brainfuck/tictactoe.ll
clang brainfuck/tictactoe.ll -Wno-override-module -o tictactoe
```

## LLVM: Cross compilation of brainfuck code on mac

On a mac you can cross compile to arm64 and/or amd64 binaries doing the following:

### Specify AMD64/x86_64 output

```bash
bfcompile -o -g llvm -c brainfuck/tictactoe.bf > brainfuck/tictactoe.ll
clang brainfuck/tictactoe.ll -Wno-override-module -target x86_64-apple-darwin-macho -o tictactoe_amd64
```

### Specify ARM64 output

```bash
bfcompile -o -g llvm -c brainfuck/tictactoe.bf > brainfuck/tictactoe.ll
clang brainfuck/tictactoe.ll -Wno-override-module -target arm64-apple-darwin-macho -o tictactoe_arm64
```

## QBE: Cross compilation of brainfuck code on mac

On a mac you can cross compile to arm64 and/or x86_64 binaries doing the following:

### Specify AMD64/x86_64 output

```bash
./bfcompile -o -g qbe -c brainfuck/tictactoe.bf > brainfuck/tictactoe.ssa
qbe -t amd64_apple -o brainfuck/tictactoe.s brainfuck/tictactoe.ssa
cc brainfuck/tictactoe.s -target x86_64-apple-darwin-macho -o tictactoe_amd64
```

### Specify ARM64 output

```bash
./bfcompile -o -g qbe -c brainfuck/tictactoe.bf > brainfuck/tictactoe.ssa
qbe -t arm64_apple -o brainfuck/tictactoe.s brainfuck/tictactoe.ssa
cc brainfuck/tictactoe.s -target arm64-apple-darwin-macho -o tictactoe_arm64
```
