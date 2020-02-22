package main

import (
	"fmt"
	"log"
	"os"
)

var (
	warnLogger  = log.New(os.Stdout, "[WARN]: ", log.Lshortfile)
	infoLogger  = log.New(os.Stdout, "[INFO]: ", log.LstdFlags)
	fatalLogger = log.New(os.Stdout, "[FATAL]: ", log.Llongfile)
)

// LogWarn logs warning messages.
func LogWarn(v ...interface{}) {
	warnLogger.Output(2, fmt.Sprintln(v...))
}

// LogWarnf logs warning messages with a format string.
func LogWarnf(format string, v ...interface{}) {
	warnLogger.Output(2, fmt.Sprintf(format, v...))
}

// LogInfo logs info messages.
func LogInfo(v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintln(v...))
}

// LogInfof logs info messages with a format string.
func LogInfof(format string, v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintf(format, v...))
}

// LogFatal logs fatal messages and exits.
func LogFatal(v ...interface{}) {
	fatalLogger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

// LogFatalf logs fatal messages with a format string.
func LogFatalf(format string, v ...interface{}) {
	fatalLogger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
