// Package log enables controlling gitfs logging.
package log

type Logger interface {
	Printf(format string, v ...interface{})
}

var Log Logger = nil

func Printf(format string, v ...interface{}) {
	if Log == nil {
		return
	}
	Log.Printf(format, v...)
}
