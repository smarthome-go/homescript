# The Homescript Semantic Fuzzer

## Goals

The main goal of the semantic fuzzer is to break the compiler so that these bugs can be fixed.

What counts as compiler / VM misbehavior?

- The compiler crashes / panics
- Two semantically equivalent programs produce different outputs

## How It Works

The fuzzer will take an analyzed tree as its input in order to transform it into another tree that has the same semantic meaning (on a high level).
However, the underlying structure of the program will be changed in a way that preserves the output of the program.
For instance, a `if-else` might be represented using `match` instead.

## Roadmap

- Writing an initial demo
- Adding recursion / output limits
- Integrating the fuzzer into the automated testing suite
