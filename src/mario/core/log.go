//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Mario logging.
//
package core

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	writer      io.Writer
	cache       string
	enableCache bool
}

var ENABLE_LOGGING bool = true
var LOGGER *Logger = nil

func logInit() {
	if LOGGER == nil {
		LOGGER = &Logger{io.Writer(os.Stdout), "", false}
	}
}

func logWrite(msg string) {
	logInit()
	LOGGER.writer.Write([]byte(msg))
	if LOGGER.enableCache {
		LOGGER.cache += msg
	}
}

func LogEnableCache() {
	if ENABLE_LOGGING {
		logInit()
		LOGGER.enableCache = true
		LOGGER.cache = ""
	}
}

func LogDisableCache() {
	if ENABLE_LOGGING {
		logInit()
		LOGGER.enableCache = false
		LOGGER.cache = ""
	}
}

func LogTee(filename string) {
	if ENABLE_LOGGING {
		logInit()
		f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
		LOGGER.writer = io.MultiWriter(LOGGER.writer, f)
		f.WriteString(LOGGER.cache)
	}
}

func Log(format string, v ...interface{}) {
	if ENABLE_LOGGING {
		logWrite(fmt.Sprintf(format, v...))
	}
}

func LogInfo(component string, format string, v ...interface{}) {
	if ENABLE_LOGGING {
		logWrite(fmt.Sprintf("%s [%s] %s\n", Timestamp(), component, fmt.Sprintf(format, v...)))
	}
}

func LogError(err error, component string, format string, v ...interface{}) {
	if ENABLE_LOGGING {
		logWrite(fmt.Sprintf("%s [%s] %s\n          %s\n", Timestamp(), component, fmt.Sprintf(format, v...), err.Error()))
	}
}
