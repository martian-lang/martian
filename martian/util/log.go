//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian logging.
//

package util

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	stdoutWriter io.Writer
	fileWriter   io.Writer
	cache        string
}

var ENABLE_LOGGING bool = true
var LOGGER *Logger = nil

const (
	ANSI_BLACK   = 30
	ANSI_RED     = 31
	ANSI_GREEN   = 32
	ANSI_YELLOW  = 33
	ANSI_BLUE    = 34
	ANSI_MAGENTA = 35
	ANSI_CYAN    = 36
	ANSI_WHITE   = 37
)

func logInit() bool {
	if ENABLE_LOGGING {
		if LOGGER == nil {
			LOGGER = &Logger{io.Writer(os.Stdout), nil, ""}
		}
		return true
	}
	return false
}

func log(msg string) {
	if logInit() {
		if LOGGER.fileWriter != nil {
			LOGGER.fileWriter.Write([]byte(msg))
		} else {
			LOGGER.cache += msg
		}
	}
}

func print(msg string) {
	if logInit() {
		LOGGER.stdoutWriter.Write([]byte(msg))
		log(msg)
	}
}

// Sets up the logging methods to log to the given file.
func LogTee(filename string) {
	if logInit() {
		if LOGGER.fileWriter == nil {
			logInit()
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("ERROR: Could not open log file: ", err)
			} else {
				LOGGER.fileWriter = io.Writer(f)
				log(LOGGER.cache)
			}
		}
	}
}

// Sets up the logging methods to log to the given writer.
func LogTeeWriter(writer io.Writer) {
	if logInit() {
		if LOGGER.fileWriter == nil {
			logInit()
			LOGGER.fileWriter = writer
			log(LOGGER.cache)
		}
	}
}

func formatRaw(format string, v ...interface{}) string {
	return fmt.Sprintf(format, v...)
}

func formatInfo(component string, format string, v ...interface{}) string {
	return fmt.Sprintf("%s [%s] %s\n", Timestamp(), component, fmt.Sprintf(format, v...))
}

func formatError(err error, component string, format string, v ...interface{}) string {
	return fmt.Sprintf("%s [%s] %s\n          %s\n", Timestamp(), component, fmt.Sprintf(format, v...), err.Error())
}

// Logs the given string to the current log stream.  If one has not been
// initialized with LogTee or LogTeeWriter, the content is buffered until
// one of those methods is called.
func Log(format string, v ...interface{}) {
	log(formatRaw(format, v...))
}

// Logs a line in the form "<timestamp> [component] <message>".  Component
// is intended to indicate the source of the message and should be a consistent
// length.  By convention, this length is 7 characters.
func LogInfo(component string, format string, v ...interface{}) {
	log(formatInfo(component, format, v...))
}

// Logs a line in the form "<timestamp> [component] <message>" followed by
// a line with err.Error().  Component is intended to indicate the source of
// the message and should be a consistent length.  By convention, this length
// is 7 characters.
func LogError(err error, component string, format string, v ...interface{}) {
	log(formatError(err, component, format, v...))
}

// Like Log, but also prints to standard output.
func Print(format string, v ...interface{}) {
	print(formatRaw(format, v...))
}

func Println(format string, v ...interface{}) {
	print(formatRaw(format, v...) + "\n")
}

// Like LogInfo but also prints to standard output.
func PrintInfo(component string, format string, v ...interface{}) {
	print(formatInfo(component, format, v...))
}

// Like LogError but also prints to standard output.
func PrintError(err error, component string, format string, v ...interface{}) {
	print(formatError(err, component, format, v...))
}

// Surrounds the given string with ANSI color control characters.
func Colorize(s string, c int) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", c, s)
}
