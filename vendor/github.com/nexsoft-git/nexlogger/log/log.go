// Package log provides a global logger for nexlogger.
package log

import (
	"context"
	"fmt"
	"io"

	"github.com/nexsoft-git/nexlogger"
)

// Logger is the global logger.
var Logger nexlogger.Logger

func InitiateLogger(log nexlogger.Logger) {
	Logger = log
}

// Output duplicates the global logger and sets w as its output.
func Output(w io.Writer) nexlogger.Logger {
	return Logger.Output(w)
}

// With creates a child logger with the field added to its context.
func With() nexlogger.Context {
	return Logger.With()
}

// Level creates a child logger with the minimum accepted level set to level.
func Level(level nexlogger.Level) nexlogger.Logger {
	return Logger.Level(level)
}

// Sample returns a logger with the s sampler.
func Sample(s nexlogger.Sampler) nexlogger.Logger {
	return Logger.Sample(s)
}

// Hook returns a logger with the h Hook.
func Hook(h nexlogger.Hook) nexlogger.Logger {
	return Logger.Hook(h)
}

// Err starts a new message with error level with err as a field if not nil or
// with info level if err is nil.
//
// You must call Msg on the returned event in order to send the event.
func Err(err error) *nexlogger.Event {
	return Logger.Err(err)
}

// Trace starts a new message with trace level.
//
// You must call Msg on the returned event in order to send the event.
func Trace() *nexlogger.Event {
	return Logger.Trace()
}

// Debug starts a new message with debug level.
//
// You must call Msg on the returned event in order to send the event.
func Debug() *nexlogger.Event {
	return Logger.Debug()
}

// Info starts a new message with info level.
//
// You must call Msg on the returned event in order to send the event.
func Info() *nexlogger.Event {
	return Logger.Info()
}

// Warn starts a new message with warn level.
//
// You must call Msg on the returned event in order to send the event.
func Warn() *nexlogger.Event {
	return Logger.Warn()
}

// Error starts a new message with error level.
//
// You must call Msg on the returned event in order to send the event.
func Error() *nexlogger.Event {
	return Logger.Error()
}

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called by the Msg method.
//
// You must call Msg on the returned event in order to send the event.
func Fatal() *nexlogger.Event {
	return Logger.Fatal()
}

// Panic starts a new message with panic level. The message is also sent
// to the panic function.
//
// You must call Msg on the returned event in order to send the event.
func Panic() *nexlogger.Event {
	return Logger.Panic()
}

// WithLevel starts a new message with level.
//
// You must call Msg on the returned event in order to send the event.
func WithLevel(level nexlogger.Level) *nexlogger.Event {
	return Logger.WithLevel(level)
}

// Log starts a new message with no level. Setting nexlogger.GlobalLevel to
// nexlogger.Disabled will still disable events produced by this method.
//
// You must call Msg on the returned event in order to send the event.
func Log() *nexlogger.Event {
	return Logger.Log()
}

// Print sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	Logger.Debug().CallerSkipFrame(1).Msg(fmt.Sprint(v...))
}

// Printf sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	Logger.Debug().CallerSkipFrame(1).Msgf(format, v...)
}

// Ctx returns the Logger associated with the ctx. If no logger
// is associated, a disabled logger is returned.
func Ctx(ctx context.Context) *nexlogger.Logger {
	return nexlogger.Ctx(ctx)
}
