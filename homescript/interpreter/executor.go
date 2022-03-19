package interpreter

type NotificationLevel uint8

const (
	LevelInfo NotificationLevel = iota
	LevelWarn
	LevelError
)

type Executor interface {
	Sleep(seconds int)
	Print(args ...string)
	SwitchOn(name string) bool
	Switch(name string, on bool)
	Play(server string, mode string)
	Notify(title string, description string, level NotificationLevel)

	// Builtin variables
	GetUser() string
	GetWeather() string
	GetTemperature() int
	GetDate() (int, int, int, int, int, int)
}
