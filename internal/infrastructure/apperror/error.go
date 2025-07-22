package apperror

// Error is used for all application-level errors that should be shown to the user (e.g. 400, 401, 403).
// For internal server errors (500), use fmt.Errorf and handle them separately to avoid exposing internal details to the client.
type Error struct {
	Message  string   `json:"message"`
	Code     code     `json:"code"`
	LogLevel logLevel `json:"log_level"`
}

type code int

const (
	BadRequest   code = 1
	NotFound     code = 2
	Unauthorized code = 3
	Forbidden    code = 4
)

type logLevel int

const (
	LogLevelError logLevel = 0
	LogLevelWarn  logLevel = 1
)

func (e *Error) Error() string {
	return e.Message
}
