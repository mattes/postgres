# Postgres [![GoDoc](https://godoc.org/github.com/mattes/postgres?status.svg)](https://godoc.org/github.com/mattes/postgres)

Postgres wraps [github.com/lib/pq](https://github.com/lib/pq) and implements
a set of new features on top of it:

* [Get](https://godoc.org/github.com/mattes/postgres#example-Postgres-Get),
  [Filter](https://godoc.org/github.com/mattes/postgres#example-Postgres-Filter)
  (with [untrusted input](https://godoc.org/github.com/mattes/postgres#example-Postgres-Filter-UntrustedQuery)),
  [Insert](https://godoc.org/github.com/mattes/postgres#example-Postgres-Insert), 
  [Update](https://godoc.org/github.com/mattes/postgres#example-Postgres-Update), 
  [Save](https://godoc.org/github.com/mattes/postgres#example-Postgres-Save) and 
  [Delete](https://godoc.org/github.com/mattes/postgres#example-Postgres-Delete)
  convenience functions
* Advanced encoding & decoding between Go and Postgres column types
* Create tables, indexes and foreign keys for Go structs

Postgres >= 11 required.

__Status:__ under active development, exposed func signatures mostly stable


## Usage

```go
import pg "github.com/mattes/postgres"

type User struct {
  Id    string `db:"pk"`
  Name  string
  Email string
}

func init() {
  // Register struct &User{} with alias 'user.v1',
  // and have new ids prefixed with 'user'.
  pg.RegisterWithPrefix(&User{}, "user.v1", "user")
}

func main() {
  db, _ := pg.Open(os.Getenv("DB"))
  db.Migrate(context.Background())

  u := &User{
    Id:    pg.NewID(&User{}), // example: user_1R0D8rn6jP870lrtSpgb1y6M5tG
    Name:  "Karl",
    Email: "karl@example.com",
  }
  db.Insert(context.Background(), u) // insert into table user_v1

  // ... for more examples, have a look at the docs.
}
```

### Encoding & Decoding of Go types

This package converts between the following types. A postgres column
can be null, if the related Go type can be nil. Otherwise Go's zero value
is used as the postgres default value. Complex Go types are stored as JSON.

| Go type                         | Postgres column type            |
|---------------------------------|---------------------------------|
| implements ColumnTyper          | "returned value"                |
| implements sql.Scanner          | text null                       |
| time.Time                       | timestamp (6) without time zone |
| time.Duration                   | bigint                          |
| []string                        | text[] null                     |
| string                          | text not null default ''        |
| bool                            | boolean not null default false  |
| int                             | integer not null default 0      |
| struct{}                        | jsonb null                      |
| []T                             | jsonb null                      |
| map[T]T                         | jsonb null                      |


### Migrations for Go structs

Postgres tables should follow Go structs. This package is able to automatically
run migrations to create new tables with primary keys, indexes and foreign keys.

Only backwards compatible, non-destructive migrations are applied to ensure
that two or more Go processes with different Go struct schemas can run at the same time.

| Use-case                                     | automatically migrated?                                               |
|----------------------------------------------|-----------------------------------------------------------------------|
| Create a new table for struct                | Yes                                                                   |
| Add a new column for a new field in a struct | Yes                                                                   |
| Change field's name                          | No                                                                    |
| Change field's type                          | No                                                                    |
| Remove a field                               | No                                                                    |
| Add a new field to primary key               | No                                                                    |
| Remove a field from primary key              | No                                                                    |
| Add a new index                              | Yes                                                                   |
| Remove an index                              | No                                                                    |
| Add a new field to index                     | No                                                                    |
| Remove a field from index                    | No                                                                    |
| Add a new unique index                       | Yes, if existing data doesn't violate unique constraint.              |
| Remove an unique index                       | No                                                                    |
| Add a new field to unique index              | No                                                                    |
| Remove a field from unique index             | No                                                                    |
| Add a new foreign key                        | Yes, if existing data doesn't violate unique/ foreign key constraint. |
| Remove a foreign key                         | No                                                                    |

Changes that are not backwards compatible usually require all deprecated Go processes to stop
first. To enable zero-downtime deploys, it's recommended to either create a new table or field
and write and read from the old and new table or field simultaneously until the deprecated
versions are stopped and removed.

## Struct tags

This package will pick up `db` struct tags to build queries and create migrations. The following struct tags are supported:

### Primary Keys

```go
// Column becomes primary key
Col string `db:"pk"` 

// Col1 and Col2 become composite primary key
Col1 string `db:"pk(name=mypk, method=hash, order=desc, composite=[Col2]"` 
Col2 string
```

### Foreign Keys

```go
// Column references A.Col
Col string `db:"references(struct=A, field=Col)"`

// Column references A.Col1 and A.Col2
Col string `db:"references(struct=A, fields=[Col1, Col2])"`
```

### Indexes

```go
// Column has index
Col string `db:"index"`

// Column has composite index
Col1 string `db:"index(name=myindex, method=hash, order=desc, composite=[Col2]"`
Col2 string

// Column has unique index
Col string `db:"unique"`

// Column has unique composite index
Col1 string `db:"unique(name=myindex, method=hash, order=desc, composite=[Col2]"`
Col2 string
```

### Table Partitions

Partitions table by range, see [docs](https://www.postgresql.org/docs/11/ddl-partitioning.html).

```go
CreatedAt time.Time `db:"pk,partitionByRange"`
```

## Testing

Set env variable `GOTEST_POSTGRES_URI` (see [helper_test.go](helper_test.go) for example) 
and run `make test`.

