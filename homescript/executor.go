package homescript

import (
	"net/http"
	"time"
)

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

type ReminderUrgency uint8

const (
	UrgencyLow ReminderUrgency = iota
	UrgencyNormal
	UrgencyMedium
	UrgencyHigh
	UrgencyUrgent
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
	Cookies    []http.Cookie
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
	Remind(title string, description string, urgency ReminderUrgency, dueDate time.Time) (uint, error)
	Log(title string, description string, level LogLevel) error
	Exec(homescriptId string, args map[string]string) (ExecResponse, error)
	ResolveModule(homescriptId string) (code string, filename string, found bool, shouldProceed bool, err error)
	ReadFile(path string) (code string, err error)
	Get(url string) (HttpResponse, error)
	Http(url string, method string, body string, headers map[string]string) (HttpResponse, error)
	GetStorage(key string) (*string, error)
	SetStorage(key string, value string) error
	IsAnalyzer() bool

	// Builtin variables
	GetUser() string
	GetWeather() (Weather, error)
}
