package interpreter

type NotificationLevel uint8

const (
	LevelInfo NotificationLevel = iota
	LevelWarn
	LevelError
)

type Executor interface {
	Switch(name string, on bool)
	SwitchOn(name string) bool
	Sleep(seconds int)
	Print(args ...string)
	Notify(title string, description string, level NotificationLevel)
	Play(server string, mode string)
}
