package log

import (
	"io"
	"log"
	"os"
)

var DefaultFlag = log.Ldate | log.Lshortfile | log.Lmicroseconds

var std = New(os.Stderr, LevelInfo, DefaultFlag)

func Default() *Logger {
	return std
}

func SetFlags(flags int) {
	std.SetFlags(flags)
}

func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

func Error(format string, v ...any) {
	std.Info(format, v...)
}

func Warn(format string, v ...any) {
	std.Warn(format, v...)
}

func Info(format string, v ...any) {
	std.Info(format, v...)
}

func Debug(format string, v ...any) {
	std.Debug(format, v...)
}
