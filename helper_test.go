package postgres

import (
	"bytes"
	"encoding/base64"
	njson "encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	postgresURI  string
	naughtyBytes [][]byte
)

func init() {
	// load GOTEST_POSTGRES_URI from env
	postgresURI = os.Getenv("GOTEST_POSTGRES_URI")
	if postgresURI == "" {
		postgresURI = "postgres://postgres:postgres@localhost:5432/go_postgres_test?sslmode=disable&createTempTables=true"
	}

	// load naughty bytes
	naughtyBytes = loadNaughtyBytes("test_data/naughty.b64.json")

	// initialize database connection and User{} for example_test.go
	Register(&User{}, "user", "u")
}

type stdoutLogger struct{}

func (l *stdoutLogger) Query(query string, duration time.Duration, args ...interface{}) {
	fmt.Println(query)
}

func print() *stdoutLogger {
	return &stdoutLogger{}
}

type testLogger struct {
	writer bytes.Buffer
	debug  bool
}

var spewQueryArgsConfig = &spew.ConfigState{
	Indent:                  "  ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
}

func (l *testLogger) Query(query string, duration time.Duration, args ...interface{}) {
	var q string
	if len(args) > 0 {
		q = fmt.Sprintf("%v ... with args:\n%v", query, spewQueryArgsConfig.Sdump(args))
	} else {
		q = query
	}

	_, err := l.writer.Write(append([]byte(q), []byte("\n\n")...))
	if err != nil {
		panic(err)
	}

	if l.debug {
		fmt.Println(q + "\n")
	}
}

func (l *testLogger) Write(t *testing.T, filename string) {
	t.Logf("WARNING: writing file: %v", filename)
	require.NoError(t, ioutil.WriteFile(filename, l.writer.Bytes(), 0664))
}

func (l *testLogger) Equal(t *testing.T, filename string) {

	// uncomment to quickly re-write all test files,
	// commit changes before doing this!
	// l.Write(t, filename)

	expect, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	require.Equal(t, string(expect), l.writer.String())
}

func loadNaughtyBytes(filename string) [][]byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	// read json file
	b64 := make([]string, 0)
	d := njson.NewDecoder(f)
	if err := d.Decode(&b64); err != nil {
		panic(err)
	}

	// decode base64
	o := make([][]byte, len(b64))
	for i := 0; i < len(b64); i++ {
		b, err := base64.StdEncoding.DecodeString(b64[i])
		if err != nil {
			panic(err)
		}
		o[i] = b
	}

	return o
}

func mustNewMetaStruct(v interface{}) *metaStruct {
	r, err := newMetaStruct(v)
	if err != nil {
		panic(err)
	}
	return r
}

func requirePQError(t *testing.T, err error, codeName string) {
	require.IsType(t, &pq.Error{}, err)
	require.Equal(t, codeName, err.(*pq.Error).Code.Name())
}
