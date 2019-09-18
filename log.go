package postgres

import "time"

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
