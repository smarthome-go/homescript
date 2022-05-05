# The Homescript DSL

Homescript is a fast and custom DSL (domain-specific language) for the  [Smarthome server](https://github.com/MikMuellerDev/smarthome) and the [Homescript CLI](https://github.com/MikMuellerDev/homescript-cli).
It provides a scripting interface for Smarthome users in order to create customized routines and workflows.


## Documentation
### Builtin variables
#### Weather
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
#### Temperature
```python
print(temperature)
```
The current temperature in the area of the Smarthome-server can be queried with the code above.

*Note: the temperature's measurement unit is dependent on the host implementation*

#### Time
```python
print(currentYear)
print(currentMonth)
print(currentDay)
print(currentHour)
print(currentMinute)
print(currentSecond)
```
Returns the values matching the local time of the Smarthome-server

#### User
```python
print(user)
```
Prints the username of the user currently running the script

### Builtin Functions

#### Switch
##### Changing power of a switch
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

##### Querying power of a switch
```python
print(switchOn("switchName"))
```
The code above should return the power state of the requested switch as a boolean.
#### Sending Notifications
```python
notify("Notification Title", "An interesting description", 1)
```

Notify sends a push-notification to the current user.

Legal notification levels (*last parameter*) are:
- 1 Info
- 2 Warn
- 3 Error

#### Users
##### Add User
Creates a new user with included metadata
```python
addUser('username', 'password', 'forename', 'surname')
```
##### Delete User
Deletes a user and all their data
```python
delUser('username')
```
##### Add Permission
Adds an permission to an arbitrary user
```python
addPerm('username', 'permission')
```

##### Delete Permission
Removes a permission from an arbitrary user
```python
delPerm('username', 'permission')
```

#### Logging
```python
log("Log Title", "What happened?", 4)
```
Logs a message to the server's console and to the internal logging system.

Depending on the implementation, this should only be allowed to the admin user.
Legal log levels (*last parameter*) are:
- 0 Trace
- 1 Debug
- 2 Info
- 3 Warn
- 4 Error
- 5 Fatal

#### RadiGo
```python
play("server id", "mode id")
```
If Smarthome is used with a [RadiGo](https://github.com/MikMuellerDev/radiGo) server, Homescript can change the modes.

#### Exit
```python
exit(42)
```
Exit stops execution of the running script with a provided exit code.
Any non-0 exit code indicates a failure.
## A possible Homescript script

```python
# This project was developed for Smarthome
# https://github.com/MikMuellerDev/smarthome

# `If` is an expression and can therefore be used inline
switch(if temperature > 10 { 'switch1' } else { 'switch2' }, off)

# All built-ins are provided by Smarthome, allowing interaction between Homescript and Smarthome
# Changes power for said outlet, on / off are aliases for true / false
switch('switch3', true)

# There are some built-in variables which are also provided by Smarthome during runtime:
# Simple concatenation in print is supported
print("The current temperature is ", temperature, " degrees.")
print(weather)
print(currentHour)
print(user)

# `switchOn` can be used to query the power state of a given switch
if switchOn('switch3')  {
    print("switch is on.")
} else {
    print("switch is off.")
}

# `sleep` pauses the execution of the current script for the given amount of seconds
sleep(1)

# `notify` sends a notification to the current user, last parameter is the level (1..3)
notify('title', 'description', 1)

# `addUser` creates a new user
addUser('username', 'password', 'forename', 'surname')

# Allows Smarthome to communicate with RadiGo servers
# https://github.com/MikMuellerDev/radiGo
play('server', 'mode')

# Exit terminates execution
exit(0)

print("Unreachable code.")
```


