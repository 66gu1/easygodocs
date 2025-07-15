package apperror

type Error struct {
	Message  string   `json:"message"`
	Code     code     `json:"code"`
	LogLevel logLevel `json:"log_level"`
}

type code int

const (
	InternalError code = 1
	BadRequest    code = 2
)

type logLevel int

const (
	LogLevelError logLevel = 0
	LogLevelWarn  logLevel = 1
)

func (e *Error) Error() string {
	return e.Message
}
