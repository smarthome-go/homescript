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
	Http(url string, method string, body string, headers map[string]string) (string, error)

	// Builtin variables
	GetUser() string
	GetWeather() (string, error)
	GetTemperature() (int, error)
	GetDate() (int, int, int, int, int, int)
}
