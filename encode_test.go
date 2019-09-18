package postgres

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Embedded struct {
	Name string
}

type SqlType struct {
	value string
}

func (t *SqlType) Scan(v interface{}) error {
	if t != nil {
		t.value = fmt.Sprintf("%v", v)
	}

	return nil
}

func (t *SqlType) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}

	return t.value, nil
}

type TestConstType string

const TestConst TestConstType = "test_const"

const TestConst2 string = "test_const_2"

type TestTypesStruct struct {
	Id              int `db:"pk"`
	String          string
	StringSlice     []string
	StringSliceNull []string

	Const1 TestConstType
	Const2 string

	Int       int
	BoolTrue  bool
	BoolFalse bool

	// TODO prohibit *[]T

	IntSlice     []int
	IntSliceNull []int

	EmbeddedSlice     []Embedded
	EmbeddedSliceNull []Embedded

	Embedded            Embedded
	EmbeddedZero        Embedded
	EmbeddedNull        *Embedded
	EmbeddedPointer     *Embedded
	EmbeddedNullPointer *Embedded

	Time            time.Time
	TimeZero        time.Time
	TimeNull        *time.Time
	TimePointer     *time.Time
	TimeNullPointer *time.Time

	Duration            time.Duration
	DurationZero        time.Duration
	DurationNull        *time.Duration
	DurationPointer     *time.Duration
	DurationNullPointer *time.Duration

	Map     map[string]string
	MapNull map[string]string

	SqlType            SqlType
	SqlTypePointer     *SqlType
	SqlTypeNullPointer *SqlType
}

func TestEncoding(t *testing.T) {
	// please note that postgres only stores microsecond 1e+6 precision
	ts := time.Date(2019, 7, 19, 16, 8, 30, 123456789, time.FixedZone("UTC+1", 60*60))
	ts1 := time.Date(2019, 7, 19, 16, 8, 30, 123456789, time.UTC)
	ts2 := time.Date(2019, 7, 19, 16, 8, 30, 123456789, time.UTC).Truncate(time.Microsecond)

	td := 7 * time.Second

	tstruct := TestTypesStruct{
		Id:                  1, // is primary key
		String:              "string",
		Const1:              TestConst,
		Const2:              TestConst2,
		Int:                 1,
		BoolTrue:            true,
		BoolFalse:           false,
		StringSlice:         []string{"foo", "bar"},
		StringSliceNull:     nil,
		IntSlice:            []int{1, 2},
		IntSliceNull:        nil,
		EmbeddedSlice:       []Embedded{{"Name"}},
		EmbeddedSliceNull:   nil,
		Embedded:            Embedded{"Name"},
		EmbeddedZero:        Embedded{},
		EmbeddedNull:        nil,
		EmbeddedPointer:     &Embedded{"Name"},
		EmbeddedNullPointer: nil,
		Time:                ts,
		TimeZero:            time.Time{},
		TimeNull:            nil,
		TimePointer:         &ts1,
		TimeNullPointer:     nil,
		Duration:            5 * time.Second,
		DurationZero:        time.Duration(0),
		DurationNull:        nil,
		DurationPointer:     &td,
		DurationNullPointer: nil,
		Map:                 map[string]string{"Foo": "Bar"},
		MapNull:             nil,
		SqlType:             SqlType{"foobar"},
		SqlTypePointer:      &SqlType{"foobar"},
		SqlTypeNullPointer:  nil,
	}

	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table for it
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestTypesStruct{})))

	// save data to postgres
	err = db.Save(context.Background(), &tstruct, nil)
	require.NoError(t, err)

	// get data from postgres and compare to original
	tstructx := TestTypesStruct{
		Id: 1, // get by primary key
	}
	err = db.Get(context.Background(), &tstructx)
	require.NoError(t, err)

	assert.Equal(t, tstruct, tstructx)
	assert.Equal(t, 1, tstructx.Id)
	assert.Equal(t, "string", tstructx.String)
	assert.Equal(t, TestConst, tstructx.Const1)
	assert.Equal(t, "test_const_2", tstructx.Const2)
	assert.Equal(t, 1, tstructx.Int)
	assert.Equal(t, true, tstructx.BoolTrue)
	assert.Equal(t, false, tstructx.BoolFalse)
	assert.Equal(t, []string{"foo", "bar"}, tstructx.StringSlice)
	assert.Equal(t, []string(nil), tstructx.StringSliceNull)
	assert.Equal(t, []int{1, 2}, tstructx.IntSlice)
	assert.Equal(t, []int(nil), tstructx.IntSliceNull)
	assert.Equal(t, []Embedded{{"Name"}}, tstructx.EmbeddedSlice)
	assert.Equal(t, []Embedded(nil), tstructx.EmbeddedSliceNull)
	assert.Equal(t, Embedded{"Name"}, tstructx.Embedded)
	assert.Equal(t, Embedded{}, tstructx.EmbeddedZero)
	assert.Equal(t, (*Embedded)(nil), tstructx.EmbeddedNull)
	assert.Equal(t, &Embedded{"Name"}, tstructx.EmbeddedPointer)
	assert.Equal(t, (*Embedded)(nil), tstructx.EmbeddedNullPointer)
	assert.Equal(t, time.Date(2019, 7, 19, 15, 8, 30, 123456000, time.UTC), tstructx.Time)
	assert.Equal(t, time.Time{}, tstructx.TimeZero)
	assert.Equal(t, (*time.Time)(nil), tstructx.TimeNull)
	assert.Equal(t, (*time.Time)(nil), tstructx.TimeNullPointer)
	assert.Equal(t, &ts2, tstructx.TimePointer)
	assert.Equal(t, 5*time.Second, tstructx.Duration)
	assert.Equal(t, time.Duration(0), tstructx.DurationZero)
	assert.Equal(t, (*time.Duration)(nil), tstructx.DurationNull)
	assert.Equal(t, &td, tstructx.DurationPointer)
	assert.Equal(t, (*time.Duration)(nil), tstructx.DurationNullPointer)
	assert.Equal(t, map[string]string{"Foo": "Bar"}, tstructx.Map)
	assert.Equal(t, map[string]string(nil), tstructx.MapNull)
	assert.Equal(t, SqlType{"foobar"}, tstructx.SqlType)
	assert.Equal(t, &SqlType{"foobar"}, tstructx.SqlTypePointer)
	assert.Equal(t, (*SqlType)(nil), tstructx.SqlTypeNullPointer)

	log.Equal(t, "test_data/test_encoding.txt")
}
