# Notes on HMS

- Fuzz more than the compiler (lexer + parser)
- Implement exceptions in a better way (jump from the causing instruction)
- Compile to bytecode and cache it by compiling on save
- Remove all functions and address using instruction pointer instead
- Compile on demand (on boot / load from disk)
- Implement optimizations
- Unhandled Exceptions accross function calls? (Use GOTO)
