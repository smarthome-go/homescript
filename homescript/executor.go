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

type Time struct {
	Year         uint16
	Month        uint8
	CalendarWeek uint8
	CalendarDay  uint8
	WeekDayText  string
	WeekDay      uint8
	Hour         uint8
	Minute       uint8
	Second       uint8
}

type Weather struct {
	WeatherTitle       string
	WeatherDescription string
	Temperature        float64
	FeelsLike          float64
	Humidity           uint8
}

type ExecResponse struct {
	Output      string
	RuntimeSecs float64
	ReturnValue Value
	RootScope   map[string]Value
}

type HttpResponse struct {
	Status     string
	StatusCode uint16
	Body       string
}

type Executor interface {
	Sleep(float64)
	Print(args ...string)
	Println(args ...string)
	SwitchOn(name string) (bool, error)
	Switch(name string, on bool) error
	Notify(title string, description string, level NotificationLevel) error
	Log(title string, description string, level LogLevel) error
	Exec(homescriptId string, args map[string]string) (ExecResponse, error)
	ResolveModule(homescriptId string) (string, bool, error) // Returns (module code, was found, err)
	Get(url string) (HttpResponse, error)
	Http(url string, method string, body string, headers map[string]string) (HttpResponse, error)

	// Builtin variables
	GetUser() string
	GetWeather() (Weather, error)
	GetTime() Time
}
