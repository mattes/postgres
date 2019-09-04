// Package postgres wraps https://github.com/lib/pq and implements
// a set of new features on top of it:
//
//  * Get, Filter, Insert, Update, Save and Delete convenience functions
//  * Advanced encoding & decoding between Go and Postgres column types
//  * Create tables, indexes and foreign keys for Go structs
package postgres
