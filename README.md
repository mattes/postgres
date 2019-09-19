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

This package will pick up `db` struct tags to create migrations and queries for primary keys, index and foreign keys. The following struct tags are supported:

| Struct Tag | Description |
|-----------------------------------------|--------------------------------------------------------------------------------------------------------------|
| `db:"pk"` | Create a primary key. Can be used across multiple fields to create a composite primary key. |
| `db:"references(structName.fieldName)"` | Creates a foreign key. |
| `db:"index"` | Creates an index for a field. |
| `db:"index(name)"` | Creates an index with given name. Can be used across multiple fields to create a composite index. |
| `db:"unique"` | Creates a unique index for a field. |
| `db:"unique(name)"` | Creates a unique index with given name. Can be used across multiple fields to create composite unique index. |

## Testing

Set env variable `GOTEST_POSTGRES_URI` (see [helper_test.go](helper_test.go) for example) 
and run `make test`.

