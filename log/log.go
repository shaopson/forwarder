package log

import (
	"fmt"
	"io"
	"log"
)

type Level uint8

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

type Logger struct {
	level  Level
	logger *log.Logger
}

func New(outer io.Writer, level Level, flags int) *Logger {
	if level > LevelDebug {
		level = LevelDebug
	}
	logger := &Logger{
		level:  level,
		logger: log.New(outer, "", flags),
	}
	return logger
}

func (self *Logger) SetFlags(flags int) {
	self.logger.SetFlags(flags)
}

func (self *Logger) SetOutput(w io.Writer) {
	self.logger.SetOutput(w)
}

func (self *Logger) Error(format string, v ...any) {
	self.logger.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
}

func (self *Logger) Warn(format string, v ...any) {
	if self.level < LevelWarn {
		return
	}
	self.logger.Output(2, fmt.Sprintf("[WARN] "+format, v...))
}

func (self *Logger) Info(format string, v ...any) {
	if self.level < LevelInfo {
		return
	}
	self.logger.Output(2, fmt.Sprintf("[INFO] "+format, v...))
}

func (self *Logger) Debug(format string, v ...any) {
	if self.level < LevelDebug {
		return
	}
	self.logger.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
}
