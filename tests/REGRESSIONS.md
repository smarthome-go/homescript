# Regressions tests

## Known bugs

1. Refers to [this](./regression_push_clone.hms): Object values are not cloned when pushed via the old `push` instruction. A new `CloningPush` instruction was added that is used for objects. Fixed in [#f77cf69](https://github.com/smarthome-go/homescript/commit/f77cf694efc5b2dc53b7001ed7291d1cd9af5ced).
