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

func Log(format string, v ...interface{}) {
	log(formatRaw(format, v...))
}

func LogInfo(component string, format string, v ...interface{}) {
	log(formatInfo(component, format, v...))
}

func LogError(err error, component string, format string, v ...interface{}) {
	log(formatError(err, component, format, v...))
}

func Print(format string, v ...interface{}) {
	print(formatRaw(format, v...))
}

func Println(format string, v ...interface{}) {
	print(formatRaw(format, v...) + "\n")
}

func PrintInfo(component string, format string, v ...interface{}) {
	print(formatInfo(component, format, v...))
}

func PrintError(err error, component string, format string, v ...interface{}) {
	print(formatError(err, component, format, v...))
}

func Colorize(s string, c int) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", c, s)
}
