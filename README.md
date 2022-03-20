# The Homescript DSL

*'Homescript'* is a fast and custom DSL (domain-specific language) for the  [Smarthome server](https://github.com/MikMuellerDev/smarthome).

Homescript provides a scripting interface for smarthome users in order to create customized routines and workflows which can not be represented in another way easily.

Homescript works in a similar way as most interpreted languages, consisting of a *lexer*, *parser*, and *interpreter*


## Documentation
### Builtin variables
#### Weather
```go
print(weather)
```
The current weather in a human-readable format.
Possible weather types should be:
- sunny
- rainy
- cloudy
- windy

Like any other builtin, the functionality must be provided by the host-software, (in this case smarthome) via callback functions.

*This means that variable values may change*
#### Temperature
```go
print(temperature)
```
The current temperature in the area of the smarthome-server can be queried with the code above.

*Note: the temperature's measurement unit is dependent on the implementation of the callback functions*

#### Time
```go
print(currentYear)
print(currentMonth)
print(currentDay)
print(currentHour)
print(currentMinute)
print(currentSecond)
```
Returns the values matching the localtime of the smarthome-server.
Prints the local time variables

#### User
```go
print(user)
```
Prints the username of the user currently running the script

### Builtin Functions

#### Power / Switch
##### Changing power of a switch 
```go
switch("switchName", on)
switch("switchName", true)
switch("switchName", off)
switch("switchName", false)
```
Changes the power state of a given switch.
A real implementation should check following parameters
- The user's permissions and if they are allowed to interact with this switch
- Handle errors and pass them back to homescript
- The validity of the switch and return errors

##### Quering power of a switch
```go
print(switchOn("switchName"))
```
The code above should return the power state of the requested switch as a boolean.
#### Sending Notifications
```go
notify("Notification Title", "An interesting description", 1)
```

Notify sends a push-notification to the current user.

Legal notification levels (*last parameter*) are:
- 1 Info
- 2 Warn
- 3 Error

#### Logging
```go
log("Log Title", "What happened?", 4)
```
Logs a message to the server's console and to the internal loggin system.

Depending on the implementation, this should only be allowed to the admin user.
Legal log levels (*last parameter*) are:
- 0 Trace
- 1 Debug
- 2 Info
- 3 Warn
- 4 Error
- 5 Fatal

#### RadiGo
```go
play("server id", "mode id")
```
If smarthome is used with a [radiGo](https://github.com/MikMuellerDev/radiGo) server, homescript can change the modes.

#### Exit
```go
exit(42)
```
Exit can be seen as a way to signalize the failure of a script.
Any non-0 exit code indicates the failure of the current script.
However, exit can be seen as a way to terminate the current script conditionally (top-level return), for example using guard-cases.

However, due to limitations with goroutines, `exit()` only works for local testing and in the cli.

## A possible home-script

```python
# This project was developed for smarthome
# https://github.com/MikMuellerDev/smarthome

# If is a expression and can therefore be used inline
switch(if temperature > 10 { 'switch1' } else { 'switch2' }, off)

# All builtins are later provided via callback, allowing interaction of homescript and smarthome
# Changes power for said outlet, on / off are aliases for true / false

# There are some built-in variables which behave static but are provided by smarthome during runtime:
# Simple concatenation in print is supported
print("The current temperature is ", temperature, " degrees.")
print(weather)
print(currentHour)
print(user)

# Activates the following switch
switch('switch', on)

# switchOn can be used to query the power state of a given switch
if switchOn('s3')  {
    print("switch is on.")
} else {
    print("switch is off.")
}

# Sleep takes the amount of seconds to pause the execution of the current script
sleep(1)

# Print 'prints' to a specified callback (console + output)
print('message')

# Sends a notification to the current user, last parameter is the level (1..3)
notify('title', 'description', 1)

# Allows smarthome to communicate with radigo servers
# https://github.com/MikMuellerDev/radiGo
play('server', 'mode')

# Exit terminates the current homescript file
# (can be seen as a top-level return)
exit(0)

print("Unreachable code.")
```


