package protocol

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	LogLevelOff LogLevel = iota
	LogLevelInfo
	LogLevelDebug
)

type Logger struct {
	level  LogLevel
	prefix string
	std    *log.Logger
}

func NewLogger(level LogLevel, prefix string) *Logger {
	return &Logger{
		level:  level,
		prefix: prefix,
		std:    log.New(os.Stderr, "", 0),
	}
}

func ParseLogLevel(s string) LogLevel {
	switch s {
	case "off":
		return LogLevelOff
	case "debug":
		return LogLevelDebug
	default:
		return LogLevelInfo
	}
}

func (l *Logger) Infof(format string, args ...any) {
	if l.level >= LogLevelInfo {
		l.std.Printf("[%s] INFO  %s %s\n", time.Now().Format("15:04:05"), l.prefix, fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Debugf(format string, args ...any) {
	if l.level >= LogLevelDebug {
		l.std.Printf("[%s] DEBUG %s %s\n", time.Now().Format("15:04:05"), l.prefix, fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Errorf(format string, args ...any) {
	if l.level >= LogLevelInfo {
		l.std.Printf("[%s] ERROR %s %s\n", time.Now().Format("15:04:05"), l.prefix, fmt.Sprintf(format, args...))
	}
}
