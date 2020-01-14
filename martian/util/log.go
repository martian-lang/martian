//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian logging.
//

package util

import (
	"bytes"
	"fmt"
	"io"
	golog "log"
	"os"
)

// StringWriter is the interface for writers which can write
// both bytes and strings.
type StringWriter interface {
	io.Writer
	WriteString(string) (int, error)
}

type Logger struct {
	stdoutWriter StringWriter
	fileWriter   StringWriter
	cache        bytes.Buffer
}

func (logger *Logger) Write(msg []byte) (int, error) {
	if logger.fileWriter != nil {
		return logger.fileWriter.Write(msg)
	} else {
		logger.cache.Write(msg)
		return len(msg), nil
	}
}

func (logger *Logger) WriteString(msg string) (int, error) {
	if logger.fileWriter != nil {
		return logger.fileWriter.WriteString(msg)
	} else {
		logger.cache.WriteString(msg)
		return len(msg), nil
	}
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
			LOGGER = &Logger{stdoutWriter: os.Stdout}
		}
		return true
	}
	return false
}

// Wrappers which handle lazy init and redirection to the global LOGGER.

// A writer which can be passed to fmt.Fprintf for the print methods.
type printTarget struct{}

func (p *printTarget) Write(msg []byte) (int, error) {
	if logInit() {
		LOGGER.stdoutWriter.Write(msg)
		return LOGGER.Write(msg)
	} else {
		return len(msg), nil
	}
}

func (p *printTarget) WriteString(msg string) (int, error) {
	if logInit() {
		LOGGER.stdoutWriter.WriteString(msg)
		return LOGGER.WriteString(msg)
	} else {
		return len(msg), nil
	}
}

var printWriter = printTarget{}

// A writer which can be passed to fmt.Fprintf for the log methods.
type logTarget struct{}

func (p *logTarget) Write(msg []byte) (int, error) {
	if logInit() {
		return LOGGER.Write(msg)
	} else {
		return len(msg), nil
	}
}

func (p *logTarget) WriteString(msg string) (int, error) {
	if logInit() {
		return LOGGER.WriteString(msg)
	} else {
		return len(msg), nil
	}
}

var logWriter = new(logTarget)

// Wraps the martian logger as go log.Logger object for use with, for example,
// net/http.HttpServer.ErrorLog
func GetLogger(component string) (*golog.Logger, bool) {
	if logInit() {
		return golog.New(LOGGER, "["+component+"]", golog.LstdFlags), true
	} else {
		return nil, false
	}
}

// Sets the target for Print* logging.
func SetPrintLogger(w StringWriter) {
	if ENABLE_LOGGING {
		if LOGGER == nil {
			LOGGER = &Logger{stdoutWriter: w}
		} else {
			LOGGER.stdoutWriter = w
		}
	}
}

// If the logger has an open file, check that the file exists at the expected
// location on disk, and that the file that is at that location is the same as
// the one that is open.
func VerifyLogFile() error {
	if LOGGER == nil {
		return nil
	}
	w := LOGGER.fileWriter
	if w == nil {
		return nil
	}
	f, ok := w.(*os.File)
	if !ok {
		return nil
	}
	if f.Name() == "" {
		return nil
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if currentInfo, err := os.Stat(f.Name()); err != nil {
		return err
	} else if !os.SameFile(info, currentInfo) {
		return fmt.Errorf(
			"file %q is not the same file as was previously opened.",
			f.Name())
	}
	return nil
}

// Sets up the logging methods to log to the given file.
func LogTee(filename string) {
	if logInit() {
		if LOGGER.fileWriter == nil {
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("ERROR: Could not open log file: ", err)
			} else {
				LOGGER.fileWriter = f
				LOGGER.cache.WriteTo(f)
				LOGGER.cache = bytes.Buffer{}
			}
		}
	}
}

// Sets up the logging methods to log to the given writer.
func LogTeeWriter(writer StringWriter) {
	if logInit() {
		if LOGGER.fileWriter == nil {
			LOGGER.fileWriter = writer
			LOGGER.cache.WriteTo(writer)
			LOGGER.cache = bytes.Buffer{}
		}
	}
}

func formatInfo(w io.Writer, component string, format string, v ...interface{}) {
	fmt.Fprintf(w, "%s [%s] %s\n", Timestamp(), component, fmt.Sprintf(format, v...))
}

func formatError(w io.Writer, err error, component string, format string, v ...interface{}) {
	args := make([]interface{}, 0, 3+len(v))
	args = append(args, Timestamp(), component)
	args = append(args, v...)
	args = append(args, err.Error())
	fmt.Fprintf(w, "%s [%s] "+format+"\n          %s\n",
		args...)
}

// Logs the given string to the current log stream.  If one has not been
// initialized with LogTee or LogTeeWriter, the content is buffered until
// one of those methods is called.
func Log(format string, v ...interface{}) {
	fmt.Fprintf(logWriter, format, v...)
}

// Logs a line in the form "<timestamp> [component] <message>".  Component
// is intended to indicate the source of the message and should be a consistent
// length.  By convention, this length is 7 characters.
func LogInfo(component string, format string, v ...interface{}) {
	formatInfo(logWriter, component, format, v...)
}

// Logs a line in the form "<timestamp> [component] <message>" followed by
// a line with err.Error().  Component is intended to indicate the source of
// the message and should be a consistent length.  By convention, this length
// is 7 characters.
func LogError(err error, component string, format string, v ...interface{}) {
	formatError(logWriter, err, component, format, v...)
}

// Like Log, but also prints to standard output.
func Print(format string, v ...interface{}) {
	fmt.Fprintf(&printWriter, format, v...)
}

func PrintBytes(msg []byte) {
	printWriter.Write(msg)
}

func Println(format string, v ...interface{}) {
	Print(format+"\n", v...)
}

// Like LogInfo but also prints to standard output.
func PrintInfo(component string, format string, v ...interface{}) {
	formatInfo(&printWriter, component, format, v...)
}

// Like LogError but also prints to standard output.
func PrintError(err error, component string, format string, v ...interface{}) {
	formatError(&printWriter, err, component, format, v...)
}
