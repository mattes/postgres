package postgres

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// QueryLimit sets the default LIMIT clause
	QueryLimit = 10

	// MaxQueryLength is the max size of a query string
	MaxQueryLength = 1000
)

// QueryStmt is a query builder, used by Postgres.Filter
type QueryStmt struct {
	query          string
	args           []interface{}
	order          []string
	limit          int
	untrusted      bool
	fieldWhitelist []string
}

// Query builds a query statement that will use prepared statements.
func Query(query string, args ...interface{}) *QueryStmt {
	return &QueryStmt{
		query: query,
		args:  args,
		limit: QueryLimit,
	}
}

// UntrustedQuery builds a query statement that will use prepared statements.
// The given query is verified to be valid SQL to prevent SQL injections
// and accepts untrusted user input, i.e. from URL query parameters.
//
// QueryStmt.Whitelist is required to whitelist queryable fields.
//
// The given query must follow a `field operator wildcard` syntax, i.e.
//   Email like $1 and (Active != $2 or Admin = $3)
//
// The following operators are allowed:
//   =|!=|<|<=|>|>=|ilike|like
func UntrustedQuery(untrustedQuery string, args ...interface{}) *QueryStmt {
	q := Query(untrustedQuery, args...)
	q.untrusted = true
	return q
}

// Whitelist sets acceptable fields that can be queried.
// Required when using UntrustedQuery.
func (q *QueryStmt) Whitelist(fields ...StructFieldName) *QueryStmt {
	if q.fieldWhitelist == nil {
		q.fieldWhitelist = make([]string, 0)
	}

	for _, field := range fields {
		x := toString(field)
		if x != "" {
			q.fieldWhitelist = append(q.fieldWhitelist, x)
		}
	}
	return q
}

// Limit sets the maximum number of returned query results.
func (q *QueryStmt) Limit(n int) *QueryStmt {
	if n <= 0 {
		return q // ignore
	}

	q.limit = n
	return q
}

// Asc instructs the result to be ordered ascending by field.
func (q *QueryStmt) Asc(field StructFieldName) *QueryStmt {
	q.order = append(q.order, fmt.Sprintf("%v ASC", mustIdentifier(toString(field))))
	return q
}

// Desc instructs the result to be ordered descending by field.
func (q *QueryStmt) Desc(field StructFieldName) *QueryStmt {
	q.order = append(q.order, fmt.Sprintf("%v DESC", mustIdentifier(toString(field))))
	return q
}

func (q *QueryStmt) queryStr() string {
	if q.query == "" {
		return ""
	}

	return "WHERE " + q.query
}

func (q *QueryStmt) orderStr() string {
	if len(q.order) == 0 {
		return ""
	}

	return "ORDER BY " + strings.Join(q.order, ", ")
}

func (q *QueryStmt) validate(r *metaStruct) error {
	if q.untrusted {
		return q.validateUntrusted(r)
	} else {
		return q.validateTrusted(r)
	}
}

func (q *QueryStmt) validateTrusted(r *metaStruct) error {
	if q.query == "" {
		return fmt.Errorf("empty query")
	}

	if !q.validateArgsCount() {
		return fmt.Errorf("wrong args count")
	}

	// check that parentheses are balanced, meaning number of ( and ) must be equal
	if !isParenthesesBalanced(q.query) {
		return fmt.Errorf("foo")
	}

	// if quotes " around identifiers are used, they must be balanced
	if !isQuotesBalanced(q.query) {
		return fmt.Errorf("foo")
	}

	// only verify whitelist if fields have been whitelisted
	if q.fieldWhitelist != nil {
		if !q.validateWhitelist(r.fields) {
			return fmt.Errorf("whitelist didn't match")
		}
	}

	if err := q.quoteIdentifiers(r.fields); err != nil {
		return err
	}

	return nil
}

func (q *QueryStmt) validateUntrusted(r *metaStruct) error {
	if q.query == "" {
		return fmt.Errorf("empty query")
	}

	if !q.validateArgsCount() {
		return fmt.Errorf("wrong args count")
	}

	if q.fieldWhitelist == nil {
		return fmt.Errorf("untrusted query requires whitelist")
	}

	if !isValidUntrustedQuery(q.query) {
		return fmt.Errorf("invalid query")
	}

	if !q.validateWhitelist(r.fields) {
		return fmt.Errorf("whitelist didn't match")
	}

	if err := q.quoteIdentifiers(r.fields); err != nil {
		return err
	}

	return nil
}

// validateArgsCount verifies that the number of args match the $n symbols.
// it also checks that $n symbols are consecutive, starting with 1.
func (q *QueryStmt) validateArgsCount() bool {
	consecutiveArgs := make(map[int]bool)

	matches := queryTokenRegex.FindAllStringSubmatch(q.query, -1)
	for _, parts := range matches {
		if len(parts) != 4 {
			return false
		}

		i, err := strconv.Atoi(strings.TrimLeft(parts[3], "$"))
		if err != nil {
			return false
		}

		consecutiveArgs[i] = true
	}

	if len(consecutiveArgs) != len(q.args) {
		return false
	}

	// check if args are consecutive, starting at 1
	for x := 1; x < len(q.args)+1; x++ {
		if consecutiveArgs[x] == false {
			return false
		}
	}

	return true
}

func (q *QueryStmt) validateWhitelist(r fields) bool {
	if q.fieldWhitelist == nil || len(q.fieldWhitelist) == 0 {
		return false
	}

	// confirm that whitelisted fields actually exists
	for _, x := range q.fieldWhitelist {
		found := false
		for _, n := range r.names() {
			if strings.EqualFold(toSnake(x), toSnake(n)) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	matches := queryTokenRegex.FindAllStringSubmatch(q.query, -1)
	for _, parts := range matches {
		if len(parts) != 4 {
			// be conservative here. if our regex doesn't match,
			// there might be something wrong with the query,
			// so we rather not let it through at this time.
			return false
		}

		found := false
		for _, n := range q.fieldWhitelist {
			if strings.EqualFold(toSnake(n), toSnake(parts[1])) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (q *QueryStmt) quoteIdentifiers(r fields) error {
	names := r.names()

	var replaceErr error

	q.query = queryTokenRegex.ReplaceAllStringFunc(q.query,
		func(t string) string {
			// Unfortunately Go doesn't support replacing submatches via a func,
			// see https://github.com/golang/go/issues/5690.
			// So we just split the string here and then put it back to together.

			parts := strings.SplitN(t, " ", 2)
			if len(parts) != 2 {
				return t // don't touch it if regex fails
			}

			// only replace fields we know
			for _, n := range names {
				if strings.EqualFold(toSnake(n), toSnake(parts[0])) {
					ident, err := identifier(parts[0])
					if err != nil {
						replaceErr = err
						return ""
					}
					return ident + " " + parts[1]
				}
			}

			// at least quote identifier, even if we don't know what the field is
			quoteIdent, err := quoteIdentifier(parts[0])
			if err != nil {
				replaceErr = err
				return ""
			}
			return quoteIdent + " " + parts[1]
		})

	return replaceErr
}

var (
	queryIdentifier  = `"?[a-z][a-z0-9_]*"?`       // examples: field, my_field, field3, "field"
	queryOperator    = `=|!=|<|<=|>|>=|ilike|like` // all allowed operators
	queryPlaceholder = `\$[1-9][0-9]{0,2}`         // example: $1 - $999
	queryTokenRegex  = regexp.MustCompile("(?i)(" + queryIdentifier + ") +(" + queryOperator + ") +(" + queryPlaceholder + ")")
)

func isValidUntrustedQuery(in string) bool {
	// Query should only contain identifiers and key words. We are using
	// prepared statements, so constants are replaced with dollar-sign $ notation.
	// We are not supporting some esoteric escape codes. Full spec, see:
	// https://www.postgresql.org/docs/11/sql-syntax-lexical.html

	if in == "" {
		return false
	}

	if len(in) > MaxQueryLength {
		return false
	}

	// check that parentheses are balanced, meaning number of ( and ) must be equal
	if !isParenthesesBalanced(in) {
		return false
	}

	// if quotes " around identifiers are used, they must be balanced
	if !isQuotesBalanced(in) {
		return false
	}

	// remove `identifier = $n`
	in = queryTokenRegex.ReplaceAllString(in, "")

	// TODO consider combining all the replacements into one

	// remove and & or
	in = strings.ReplaceAll(in, "and", "")
	in = strings.ReplaceAll(in, "AND", "")
	in = strings.ReplaceAll(in, "or", "")
	in = strings.ReplaceAll(in, "OR", "")

	// remove ( )
	in = strings.ReplaceAll(in, "(", "")
	in = strings.ReplaceAll(in, ")", "")

	// trim space
	in = strings.TrimSpace(in)

	return in == ""
}

func isParenthesesBalanced(in string) bool {
	open := 0

	for _, c := range in {
		if c == '(' {
			open += 1

		} else if c == ')' {
			if open == 0 {
				return false
			}
			open -= 1
		}
	}

	return open == 0
}

func isQuotesBalanced(in string) bool {
	return strings.Count(in, `"`)%2 == 0
}
