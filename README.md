# The Homescript DSL (V3)

Homescript is a custom DSL (domain-specific language) for the
[Smarthome server](https://github.com/smarthome-go/smarthome).

It provides a scripting interface for Smarthome users in order to allow them to
create customized routines and workflows.

Furthermore, more advanced users can use it to implement their own apps which
run inside Smarthome.

## Language overview

- Rust-like syntax
- Strong type-system

### Statements

#### Type Definition

```rs
type Foo = int;
type Bar = [ Foo ];
type Baz = {
  a_key: Foo,
  another_key: Bar,
};
```

#### Let Statement

```rs
let foo = "bar";
let foo: MyType = returns_any();
```

#### Return Statement

```rs
return;
return 42;
```

#### Different Loop Statements

```rs
loop {
  continue;
  break;
}

while foo < 42 {
  continue;
  break;
}

let iterator = [1, 2, 3];
// let iterator = 1..42;
// let iterator = "A String";

for i in iterator {
  println(i);
  continue;
  break;
}
```

#### Expression Statement

```rs
42;
a_call();
```

### Expressions

### Literals

```rs
fn main() {
    42;                   // integer
    3.14159265;           // float
    false;                // bool
    "A string";           // string
    null;                 // null
    none;                 // none
    1..42;                // range
    [ 1, 2, 3 ];          // list
    new { key: "Value" }; // object
    fn() -> int {};   // closure
    ( 42 );               // nested
    -1;                   // prefix
    ident = 42;           // assign
    foo();                // call
    list[-1];             // index
    foo.bar;              // member
    foo as type;          // cast

    try {                 // try-catch
      /* ... */
    } catch e {
      /* ... */
    }

    if condition {        // if-else
      /* ... */
    } else if condition {
      /* ... */
    } else {
      /* ... */
    }

    {                     // block
      /* ... */
      42
    }
}
```

### Types

| Type       | Example which yields this type | Note                                                     |
| ---------- | ------------------------------ | -------------------------------------------------------- |
| any        | `"{}".parse_json()`            | explicit type-annotations required                       |
| null       | `null`                         | stands for 'no value'                                    |
| int        | `42`                           | matches the int64 Go specification                       |
| float      | `3.14`                         | matches the float64 Go specification                     |
| bool       | `true`                         | is either `true` or `false`                              |
| string     | `"Hello World!"`               | can hold any UTF-8 character                             |
| range      | `1..42`                        | specifies a range between two `int` values               |
| list       | `[ 1, 2 ]`                     | can hold any inner type as long as all elements share it |
| any-object | `new { ? }`                    | the HMS version of a `map[string]any`                    |
| object     | `new { key: 1 }`               | no duplicate or builtin keys allowed                     |
| option     | `?42; none`                    | used for runtime handling of nullable values             |
| closure    | `fn() -> int {}`               | without an identifier, captures its environment          |

## Examples

### Calculating Fibonacci Numbers

```rs
fn fib(n: int) -> int {
    if n < 2 {
        n
    } else {
        fib(n - 1) + fib(n - 2)
    }
}

fn main() {
    for n in 2..40 {
        println(n.to_string(), "->", fib(n))
    }
}
```

### FizzBuzz

```rs
fn fizz_buzz(max: int) {
    for n in (max + 1).to_range() {
        if n % 15 == 0 {
            println("FizzBuzz");
        } else if n % 3 == 0 {
            println("Fizz");
        } else if n % 5 == 0 {
            println("Buzz");
        } else {
            println(n);
        }
    }
}

fn main() {
    fizz_buzz(15);
}
```
