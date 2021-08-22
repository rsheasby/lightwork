package lightwork

import (
	"fmt"
	"runtime"
)

// RequestLoggerBase provides the basic interface required to log at 4 simple levels.
type RequestLoggerBase interface {
	// Info will be used to log things that are useful to know, but not in any way bad.
	Info(msg string)
	// Warning will be used to log problems that are not bad enough to make the request completely fail.
	Warning(msg string)
	// Error will be used to log problems that are bad enough to make the request completely fail.
	Error(msg string)
	// WTF will be used to log problems that should absolutely never happen; in development this should panic, in production it should simply be logged.
	WTF(msg string)
	// FormatLog will be called to format logs when using the RequestLogger.*f methods.
	FormatLog(format string, values ...interface{}) (mst string)
	// WriteLogs is called once a request completes, in order to write all the logs that were created during the lifecycle of the request
	WriteLogs()
}

// RequestLogger is used to log events that occur within a request handler.
type RequestLogger struct {
	b RequestLoggerBase
}

// Info should be used to log things that are useful to know, but not in any way bad.
func (rl *RequestLogger) Info(msg string) {
	rl.b.Info(msg)
}

// Infof formats your message before logging it as Info, using the provided FormatLog function.
func (rl *RequestLogger) Infof(format string, values ...interface{}) {
	rl.Info(fmt.Sprintf(format, values...))
}

// Warning should be used to log problems that are not bad enough to make the request completely fail.
func (rl *RequestLogger) Warning(msg string) {
	rl.b.Warning(msg)
}

// Warningf formats your message before logging it as a Warning, using the provided FormatLog function.
func (rl *RequestLogger) Warningf(format string, values ...interface{}) {
	rl.Warning(fmt.Sprintf(format, values...))
}

// Error should be used to log problems that are bad enough to make the request completely fail.
func (rl *RequestLogger) Error(msg string) {
	rl.b.Error(msg)
}

// Errorf formats your message before logging it as an Error, using the provided FormatLog function.
func (rl *RequestLogger) Errorf(format string, values ...interface{}) {
	rl.Error(fmt.Sprintf(format, values...))
}

// WTF should be used to log problems that should absolutely never happen.
// This also records a stack trace as a second WTF event.
// This should be indicative of a programming bug, as opposed to an expected runtime error.
func (rl *RequestLogger) WTF(msg string) {
	rl.b.WTF(msg)
	stBuf := make([]byte, 100000)
	n := runtime.Stack(stBuf, false)
	rl.b.WTF("Stack Trace:\n" + string(stBuf[:n]))
}

// WTFf formats your message before logging it as WTF, using the provided FormatLog function.
func (rl *RequestLogger) WTFf(format string, values ...interface{}) {
	rl.WTF(fmt.Sprintf(format, values...))
}
