package postgres

// Logger is a logging interface and can be used to implement a custom logger.
type Logger interface {

	// Query will be called with the SQL query and the arguments.
	Query(query string, args ...interface{})
}

func queryLog(logger Logger, query string, args ...interface{}) {
	if logger != nil {
		logger.Query(query, args...)
	}
}
