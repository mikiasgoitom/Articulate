package logger

import (
	"log"

	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

// StdLogger is a simple logger that uses the standard log package.
type StdLogger struct{}

// NewStdLogger creates a new StdLogger.
func NewStdLogger() usecasecontract.IAppLogger {
	return &StdLogger{}
}

// Debugf logs a debug message.
func (l *StdLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// Infof logs an info message.
func (l *StdLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Warnf logs a warning message.
func (l *StdLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

// Warningf logs a warning message.
func (l *StdLogger) Warningf(format string, args ...interface{}) {
	log.Printf("[WARNING] "+format, args...)
}

// Errorf logs an error message.
func (l *StdLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Fatalf logs a fatal message and exits.
func (l *StdLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf("[FATAL] "+format, args...)
}
