// Package log enables controlling gitfs logging.
package log

import (
	stdlog "log"
	"os"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

var Log Logger = stdlog.New(os.Stderr, "[gitfs] ", stdlog.LstdFlags)

func Printf(format string, v ...interface{}) {
	if Log == nil {
		return
	}
	Log.Printf(format, v...)
}
