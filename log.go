package postgres

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// Logger is a logging interface and can be used to implement a custom logger.
type Logger interface {

	// Query will be called with the SQL query and the arguments.
	Query(query string, duration time.Duration, args ...interface{})
}

func queryLog(logger Logger, query string, duration time.Duration, args ...interface{}) {
	if logger != nil {
		logger.Query(query, duration, args...)
	}
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

type DefaultLogger struct{}

var defaultLoggerSpewArgsConfig = &spew.ConfigState{
	Indent:                  "  ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
}

func (l *DefaultLogger) Query(query string, duration time.Duration, args ...interface{}) {
	if len(args) > 0 {
		log.Printf("Query: %v [%v] with args:\n%v", query, duration, defaultLoggerSpewArgsConfig.Sdump(args))
	} else {
		log.Printf("Query: %v [%v]", query, duration)
	}
}
