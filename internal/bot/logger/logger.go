package logger

import (
	"fmt"
	"log/slog"
)

type Logger struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Logger {
	return &Logger{log: log}
}

func (l *Logger) Println(v ...interface{}) {
	l.log.Debug(fmt.Sprintln(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.log.Debug(fmt.Sprintf(format, v...))
}
