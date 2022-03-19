package interpreter

type NotificationLevel uint8

const (
	LevelInfo NotificationLevel = iota
	LevelWarn
	LevelError
)

type Executor interface {
	Exit(code int)
	Print(args ...string)
	SwitchOn(name string) (bool, error)
	Switch(name string, on bool) error
	Play(server string, mode string) error
	Notify(title string, description string, level NotificationLevel) error

	// Builtin variables
	GetUser() string
	GetWeather() (string, error)
	GetTemperature() (int, error)
	GetDate() (int, int, int, int, int, int)
}
