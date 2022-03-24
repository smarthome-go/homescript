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
	Print(args ...string)
	SwitchOn(name string) (bool, error)
	Switch(name string, on bool) error
	Play(server string, mode string) error
	Notify(title string, description string, level LogLevel) error
	Log(title string, description string, level LogLevel) error
	Exec(homescriptId string) (string, error)

	// Builtin variables
	GetUser() string
	GetWeather() (string, error)
	GetTemperature() (int, error)
	GetDate() (int, int, int, int, int, int)
	GetDebugInfo() (string, error)
}
