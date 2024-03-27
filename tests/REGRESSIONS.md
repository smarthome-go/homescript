# Regressions tests

## Known bugs

1. Refers to [this](./regression_push_clone.hms): Object values are not cloned when pushed via the old `push` instruction. A new `CloningPush` instruction was added that is used for objects. Fixed in [#f77cf69](https://github.com/smarthome-go/homescript/commit/f77cf694efc5b2dc53b7001ed7291d1cd9af5ced).

2. Refers to [this](./regression_anyobj_cast.hms): Casting `new { ? } as { ? }`, meaning any-obj to anyobj resulted in a runtime crash. Fixed in [#f73c5d4](https://github.com/smarthome-go/homescript/commit/f73c5d42ecd70d92ccc29571a30a8c9c446a7123).

3. Refers to [this](./regression_range_type.hms): Range types and range values were not printed correctly. Fixed in [#679ce0c](https://github.com/smarthome-go/homescript/commit/679ce0c985bfc3df8e587196f2d2717ea094d72c).
