# The Homescript DSL

Homescript is a fast and custom DSL (domain-specific language) for the  [Smarthome server](https://github.com/smarthome-go/smarthome) and the [Homescript CLI](https://github.com/smarthome-go/cli).

It provides a scripting interface for Smarthome users in order to create customized routines and workflows.

# Feature Documentation
## Builtin variables
### Weather
```python
print(weather)
```
The current weather in a human-readable format.
Possible weather types should be:
- sunny
- rainy
- cloudy
- windy

Like any other builtin, the functionality must be provided by the host-software.

*Note: This means that variable values may change*
### Temperature
```python
print(temperature)
```
The current temperature in the area of the Smarthome-server can be queried with the code above.

*Note: the temperature's measurement unit is dependent on the host implementation*

### Time
```python
print(currentYear)
print(currentMonth)
print(currentDay)
print(currentHour)
print(currentMinute)
print(currentSecond)
```
Returns the values matching the local time of the Smarthome-server

### User
```python
print(user)
```
Prints the username of the user currently running the script

## Builtin Functions

### Switch
#### Changing power of a switch
```python
switch("switchName", on)
switch("switchName", true)
switch("switchName", off)
switch("switchName", false)
```
Changes the power state of a given switch.
A real implementation should check following things
- The user's permissions for the switch
- The validity of the switch

#### Querying power of a switch
```python
print(switchOn("switchName"))
```

The code above should return the power state of the requested switch as a boolean value.
### Notifications
```python
notify("Notification Title", "An interesting description", 1)
```

Notify sends a push-notification to the current user.

Legal notification levels (*last parameter*) are:
- 1 Info
- 2 Warn
- 3 Error

### Users
#### Add User
Creates a new user with included metadata
```python
addUser('username', 'password', 'forename', 'surname')
```
#### Delete User
Deletes a user and all their data
```python
delUser('username')
```
#### Add Permission
Adds a permission to an arbitrary user
```python
addPerm('username', 'permission')
```
#### Delete Permission
Removes a permission from an arbitrary user
```python
delPerm('username', 'permission')
```

### Logging
```python
log("Log Title", "What happened?", 4)
```
Logs a message to the server's console and to the internal logging system.

Depending on the implementation, this should only be usable by admin users.
Legal log levels (*last parameter*) are:
- 0 Trace
- 1 Debug
- 2 Info
- 3 Warn
- 4 Error
- 5 Fatal

### HTTP
```python
print(get('http://localhost:8082'))
print(http('http://localhost:8082', 'POST', 'application/json', '{"id": 2}'))
```
As of `v0.7.0-beta`, Homescript supports the use of generic http functions.
The `get` function only accepts an arbitrary string as a url and returns the request response body as a string.

The `http` function is more generic: given a URL, a request-method, a `Content-Type`, and a body, a response body will be returned as a string.

### Exit
```python
exit(42)
```
Exit stops execution of the running script with a provided exit code.
Any non-0 exit code indicates a failure.

### Panic
```python
panic("message")
```
Panic stops execution of the running script with a provided error-message.
It is meant to be used as a way to signal the failure of a script using a known reason, for example missing arguments.

### Arguments
Call arguments can be used to control the behaviour of a Homescript dynamically.
Before accessing the value of an expected argument, it is recommended to validate that this argument
has been provided to the Homescript runtime.

#### Check Arg
For this, the *checkArg* function can be used.
The `checkArg` function returns a boolean based on whether the argument has been found or not.
```python
if checkArg('indentifier') {
    # Do something, for example accessing the argument
}
```

#### Get Arg
After validating the existence of an arbitrary argument, it can be accessed using the `getArg` function.
Just like the `checkArg` function, this requires the identifier of the desired argument.
If the argument does not exist, this function will return an error.
Due to this, it is recommended to use the `checkArg` function beforehand.

Warning: this function will always return a string because the argument type must be generic.
If the function's return value is required as an integer, it can be parsed using `num(getArg('number'))`

```python
if checkArg('indentifier') {
    print(getArg('identifier'))
}
```
#### Provide Args to Homescript call
It is common to call other Homescripts from the current code.
Sometimes you may also want to provide arguments to the Homescript to be executed.
After the required first parameter `homescript_id` of the `exec` function, additional arguments can be used as call args for the Homescript.
When providing arguments to a Homescript call, make sure to avoid duplicate key entries in order to avoid errors.
```python
exec('homescript_arg', mkArg('key', 'value'), mkArg('another_key', 'another_value'))
```

### Type Conversion
#### Parse to Number
Sometimes, for example when processing arguments, it is required to parse a string value to a number.
In this case the `num` function should be used.
The function requires one argument of type string which will then be used to attempt the type conversion.
If the function's input can not be parsed to a number, an error is returned.

```python
print(num('1'))
print(num('-1'))
print(num('+1'))
print(num('0.1'))
# Will return an error
# print(num('NaN'))
```

#### Convert to String
The ability to convert a value of any type to a textual representation or a string is just as useful as the other way around.
For this, the `str` function should be used.
The only time the `str` function can return an error is when used in conjunction with a pseudo-variable (e.g. `weather`)

```python
print(str(1))
print(str(false))
print(str(switchOn('s2')))
```

## Full example script
A full example program can be found in the [`demo.hms`](./tests/demo.hms) file

# Usage Documentation

## Implementing a Custom Executor
Due to Homescript's nature of being extensible, every function call (and some variable getters) need to be implemented by the host software.
To get started, an executor has to be implemented.
The executor acts like an interface between Homescript and the host software, providing **all** features of the language.

### Structure
The executor's signature has to match following declaration.
For a better understanding, visit the [executor](./homescript/interpreter/executor.go) source code.

```go
package interpreter

type LogLevel uint8

const (
    LevelTrace LogLevel = iota
    LevelDebug
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)

type Executor interface {
    CheckArg(identifier string) bool
    GetArg(indentifier string) (string, error)

    Sleep(float64)
    Print(args ...string)
    SwitchOn(name string) (bool, error)
    Switch(name string, on bool) error
    Notify(title string, description string, level LogLevel) error
    Log(title string, description string, level LogLevel) error
    Exec(homescriptId string, args map[string]string) (string, error)
    AddUser(username string, password string, forename string, surname string) error
    DelUser(username string) error
    AddPerm(username string, permission string) error
    DelPerm(username string, permission string) error
    Get(url string) (string, error)
    Http(url string, method string, contentType string, body string) (string, error)

    // Builtin variables
    GetUser() string
    GetWeather() (string, error)
    GetTemperature() (int, error)
    GetDate() (int, int, int, int, int, int)
}
```

### Example implementation

```go
type DummyExecutor struct{}

func (self DummyExecutor) Sleep(seconds float64) {
    time.Sleep(time.Millisecond * time.Duration(1000*seconds))
}

func (self DummyExecutor) Print(args ...string) {
    output := ""
    for _, arg := range args {
        output += arg
    }
    fmt.Println(output)
}

// More functions need to be implemented
```

## Using the Run Function
The `Run` function is used in order to start the execution of a script.
- The first argument describes the executor's definition (refer to previous)
### Example
The sample was taken from [`./main.go`](./main.go).
```go
sigTerm := make(chan int)
code, errors := homescript.Run(
    homescript.DummyExecutor{},
    "<demo>",
    "print('hello world!')"
    &sigTerm,
)
for _, err := range errors {
    // Handle the errors this way
    fmt.Println(err.Error())
}
fmt.Printf("Exit code: %d\n", code)
```
