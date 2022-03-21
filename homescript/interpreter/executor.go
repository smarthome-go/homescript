package interpreter

import "github.com/MikMuellerDev/homescript/homescript/error"

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
	Exit(code int)
	Print(args ...string)
	SwitchOn(name string) (bool, *error.Error)
	Switch(name string, on bool) *error.Error
	Play(server string, mode string) *error.Error
	Notify(title string, description string, level LogLevel) *error.Error
	Log(title string, description string, level LogLevel) *error.Error

	// Builtin variables
	GetUser() string
	GetWeather() (string, *error.Error)
	GetTemperature() (int, *error.Error)
	GetDate() (int, int, int, int, int, int)
}
