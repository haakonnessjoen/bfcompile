# Brainfuck compiler

This project parses brainfuck files, optimizes it, and generates [QBE IL](https://c9x.me/compile/doc/il.html) or C code output, ready to be compiled by the LLVM compiler.

The idea came when I saw the Youtuber [tsoding](https://www.youtube.com/watch?v=JTjNoejn4iA) try out QBE, and the language looked very simple, so I thought it should be simple to use it to compile brainfuck.

## Features

Optimizes the token tree (or array really) before it generates QBE IL/C code, by compacting multiple "+" operations to a single operation using add, etc.

## Prerequisites

Unless you are using the C output feature, you need **qbe**, on mac you can just do `brew install qbe`.
