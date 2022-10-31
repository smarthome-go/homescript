package homescript

type LogLevel uint8

const (
	LevelTrace LogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

type NotificationLevel uint8

const (
	NotiInfo NotificationLevel = iota
	NotiWarning
	NotiCritical
)

type Weather struct {
	WeatherTitle       string
	WeatherDescription string
	Temperature        float64
	FeelsLike          float64
	Humidity           uint8
}

type ExecResponse struct {
	RuntimeSecs float64
	ReturnValue Value
	RootScope   map[string]*Value
}

type HttpResponse struct {
	Status     string
	StatusCode uint16
	Body       string
}

type SwitchResponse struct {
	Name  string
	Power bool
	Watts uint
}

type Executor interface {
	Sleep(float64)
	Print(args ...string) error
	Println(args ...string) error
	GetSwitch(id string) (SwitchResponse, error)
	Switch(name string, on bool) error
	Ping(ip string, timeout float64) (bool, error)
	Notify(title string, description string, level NotificationLevel) error
	Log(title string, description string, level LogLevel) error
	Exec(homescriptId string, args map[string]string) (ExecResponse, error)
	ResolveModule(homescriptId string) (string, bool, bool, error) // Returns (module code, was found, contains userful code, err)
	Get(url string) (HttpResponse, error)
	Http(url string, method string, body string, headers map[string]string) (HttpResponse, error)

	// Builtin variables
	GetUser() string
	GetWeather() (Weather, error)
}
