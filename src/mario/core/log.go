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

var ENABLE_LOGGING bool = true
var LOGGER io.Writer = nil

func logInit() {
	if LOGGER == nil {
		LOGGER = io.Writer(os.Stdout)
	}
}

func logWrite(msg string) {
	logInit()
	LOGGER.Write([]byte(msg))
}

func LogTee(filename string) {
	if ENABLE_LOGGING {
		logInit()
		f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
		LOGGER = io.MultiWriter(LOGGER, f)
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
