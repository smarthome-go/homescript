package homescript

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type DummyExecutor struct{}

func (self DummyExecutor) ResolveModule(id string) (string, bool, bool, error) {
	path := "test/programs/" + id + ".hms"
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, false, nil
		}
		return "", false, false, fmt.Errorf("read file: %s", err.Error())
	}
	return string(file), true, true, nil
}

func (self DummyExecutor) Sleep(sleepTime float64) {
	time.Sleep(time.Duration(sleepTime * 1000 * float64(time.Millisecond)))
}

func (self DummyExecutor) Print(args ...string) error {
	fmt.Printf("%s", strings.Join(args, " "))
	return nil
}

func (self DummyExecutor) Println(args ...string) error {
	fmt.Println(strings.Join(args, " "))
	return nil
}

func (self DummyExecutor) GetSwitch(id string) (SwitchResponse, error) {
	return SwitchResponse{}, nil
}

func (self DummyExecutor) Switch(name string, power bool) error {
	return nil
}

func (self DummyExecutor) Ping(ip string, timeout float64) (bool, error) {
	return false, nil
}

func (self DummyExecutor) Notify(title string, description string, level NotificationLevel) error {
	return nil
}

func (self DummyExecutor) Remind(title string, description string, urgency ReminderUrgency, dueDate time.Time) (uint, error) {
	return 0, nil
}

func (self DummyExecutor) Log(title string, description string, level LogLevel) error {
	return nil
}

func (self DummyExecutor) Exec(id string, args map[string]string) (ExecResponse, error) {
	return ExecResponse{
		RuntimeSecs: 0.2,
		ReturnValue: ValueNull{},
	}, nil
}

func (self DummyExecutor) Get(url string) (HttpResponse, error) {
	return HttpResponse{
		Status:     "OK",
		StatusCode: 200,
		Body:       "{\"foo\": \"bar\"}",
	}, nil
}

func (self DummyExecutor) Http(url string, method string, body string, headers map[string]string) (HttpResponse, error) {
	return HttpResponse{
		Status:     "Internal Server Error",
		StatusCode: 500,
		Body:       "{\"error\": \"the server is currently running on JavaScript\"}",
	}, nil
}

func (self DummyExecutor) GetUser() string {
	return "john_doe"
}

func (self DummyExecutor) GetWeather() (Weather, error) {
	return Weather{
		WeatherTitle:       "Rain",
		WeatherDescription: "light rain",
		Temperature:        17.0,
		FeelsLike:          16.0,
		Humidity:           87,
	}, nil
}

func (self DummyExecutor) GetStorage(_ string) (*string, error) {
	s := ""
	return &s, nil
}

func (self DummyExecutor) SetStorage(key string, value string) error {
	return nil
}
