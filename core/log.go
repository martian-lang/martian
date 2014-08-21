//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario logging.
//
package core

import (
	"fmt"
)

func LogInfo(component string, format string, v ...interface{}) {
	fmt.Printf("%s [%s] %s\n", Timestamp(), component, fmt.Sprintf(format, v...))
}

func LogError(err error, component string, format string, v ...interface{}) {
	fmt.Printf("%s [%s] %s\n          %s\n", Timestamp(), component, fmt.Sprintf(format, v...), err.Error())
}
