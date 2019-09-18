package postgres

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Example struct{}

func ExamplePostgres_Migrate() {
	db, _ := Open(postgresURI)

	// Register "example" struct with prefix "ex"
	RegisterWithPrefix(&Example{}, "example", "ex")

	// Run migrations ...
	_ = db.Migrate(context.Background())
}

type User struct {
	Id    string `db:"pk"`
	Name  string
	Email string
}

func ExamplePostgres_Get() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_1", // primary key, via `db:"pk"` struct tag
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Save(context.Background(), user)

	// Get user by Id
	user = &User{
		Id: "user_1",
	}
	_ = db.Get(context.Background(), user)
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) ON CONFLICT ("id") DO UPDATE SET ("name", "email") = ROW("excluded"."name", "excluded"."email") RETURNING "id", "name", "email"
	// SELECT "id", "name", "email" FROM "user" WHERE "id" = $1 LIMIT 1
	// &{Id:user_1 Name:Peter Email:peter@foobar.com}
}

func ExamplePostgres_Filter() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_5",
		Name:  "Max",
		Email: "max@example.com",
	}
	_ = db.Save(context.Background(), user)

	// Filter users by email
	users := []User{}
	_ = db.Filter(context.Background(), &users, Query("Email LIKE $1", "%example.com"))
	fmt.Printf("%+v", users)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) ON CONFLICT ("id") DO UPDATE SET ("name", "email") = ROW("excluded"."name", "excluded"."email") RETURNING "id", "name", "email"
	// SELECT "id", "name", "email" FROM "user" WHERE "email" LIKE $1 LIMIT 10
	// [{Id:user_5 Name:Max Email:max@example.com}]
}

func ExamplePostgres_Insert() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user
	user := &User{
		Id:    "user_6", // primary key, via `db:"pk"` struct tag
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Insert(context.Background(), user)
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) RETURNING "id", "name", "email"
	// &{Id:user_6 Name:Peter Email:peter@foobar.com}
}

func ExamplePostgres_Insert_fieldMask() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user
	user := &User{
		Id:    "user_9", // primary key, via `db:"pk"` struct tag
		Email: "peter@foobar.com",
	}
	_ = db.Insert(context.Background(), user, "Id", "Email")
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "email") VALUES ($1, $2) RETURNING "id", "name", "email"
	// &{Id:user_9 Name: Email:peter@foobar.com}
}

func ExamplePostgres_Update() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_7",
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Insert(context.Background(), user)

	// Then update the user
	user = &User{
		Id:    "user_7", // primary key, via `db:"pk"` struct tag
		Name:  "Karl",
		Email: "karl@foobar.com",
	}
	_ = db.Update(context.Background(), user)
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) RETURNING "id", "name", "email"
	// UPDATE "user" SET ("name", "email") = ROW($1, $2) WHERE "id" = $3 RETURNING "id", "name", "email"
	// &{Id:user_7 Name:Karl Email:karl@foobar.com}
}

func ExamplePostgres_Update_fieldMask() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_8",
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Insert(context.Background(), user)

	// Then update the user
	user = &User{
		Id:    "user_8", // primary key, via `db:"pk"` struct tag
		Email: "karl@foobar.com",
	}
	_ = db.Update(context.Background(), user, "Email")
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) RETURNING "id", "name", "email"
	// UPDATE "user" SET ("email") = ROW($1) WHERE "id" = $2 RETURNING "id", "name", "email"
	// &{Id:user_8 Name:Peter Email:karl@foobar.com}
}

func ExamplePostgres_Save() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create or update user
	user := &User{
		Id:    "user_2", // primary key, via `db:"pk"` struct tag
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Save(context.Background(), user)
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) ON CONFLICT ("id") DO UPDATE SET ("name", "email") = ROW("excluded"."name", "excluded"."email") RETURNING "id", "name", "email"
	// &{Id:user_2 Name:Peter Email:peter@foobar.com}
}

func ExamplePostgres_Save_fieldMask() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create or update user (only save Id and Email)
	user := &User{
		Id:    "user_3", // primary key, via `db:"pk"` struct tag
		Email: "peter@foobar.com",
	}
	_ = db.Save(context.Background(), user, "Id", "Email")
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "email") VALUES ($1, $2) ON CONFLICT ("id") DO UPDATE SET ("email") = ROW("excluded"."email") RETURNING "id", "name", "email"
	// &{Id:user_3 Name: Email:peter@foobar.com}
}

func ExamplePostgres_Delete() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_4", // primary key, via `db:"pk"` struct tag
		Name:  "Peter",
		Email: "peter@foobar.com",
	}
	_ = db.Save(context.Background(), user)

	// Delete user by Id
	user = &User{
		Id: "user_4",
	}
	_ = db.Delete(context.Background(), user)
	fmt.Printf("%+v", user)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) ON CONFLICT ("id") DO UPDATE SET ("name", "email") = ROW("excluded"."name", "excluded"."email") RETURNING "id", "name", "email"
	// DELETE FROM "user" WHERE "id" = $1 RETURNING "id", "name", "email"
	// &{Id:user_4 Name:Peter Email:peter@foobar.com}
}

func ExampleTransaction_Commit() error {
	// Open connection to Postgres
	db, err := Open(postgresURI)
	if err != nil {
		return err
	}

	// Start a new transaction
	tx, err := db.NewTransaction()
	if err != nil {
		return err
	}

	// Execute SQL
	if _, err := tx.Exec(context.Background(), "query"); err != nil {
		_ = tx.Rollback() // Rollback and end the transaction
		return err        // Return the original error
	}

	// Execute more SQL
	if _, err := tx.Exec(context.Background(), "another query"); err != nil {
		_ = tx.Rollback() // Rollback and end the transaction
		return err        // Return the original error
	}

	// Commit and end the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func ExamplePostgres_Transaction() error {
	// Open connection to Postgres
	db, err := Open(postgresURI)
	if err != nil {
		return err
	}

	// Start a new transaction
	err = db.Transaction(func(tx *Transaction) error {

		// Execute SQL
		if _, err := tx.Exec(context.Background(), "query"); err != nil {
			return err
		}

		// Execute more SQL
		if _, err := tx.Exec(context.Background(), "another query"); err != nil {
			return err
		}

		return nil
	})

	// The transaction ended. If no error was returned, the transaction
	// was commited, otherwise the transaction was rolled back and the
	// original error is returned.
	return err
}

type ExampleUntrustedQuery_Struct struct {
	Email   string
	Primary bool
}

func ExamplePostgres_Filter_untrustedQuery() {
	db, _ := Open(postgresURI)
	db.Migrate(context.Background())
	db.Logger = print()

	// Create a new user first
	user := &User{
		Id:    "user_6",
		Name:  "Karl",
		Email: "karl@company.co",
	}
	_ = db.Save(context.Background(), user)

	// Simulate an incoming HTTP request, with an URL like:
	//   ?filter=Name = $1 and Email = $2
	//   &vars=Karl
	//   &vars=karl@company.co
	request := http.Request{
		URL: &url.URL{
			RawQuery: "filter=Name%20%3D%20%241%20and%20Email%20%3D%20%242&vars=Karl&vars=karl%40company.co",
		},
	}

	// Get ?filter from request URL
	urlFilter := request.URL.Query().Get("filter")

	// Get &vars from request URL and convert from []string to []interface{}
	urlVars := stringSliceToInterfaceSlice(request.URL.Query()["vars"])

	// Assemble new UntrustedQuery from URL input
	query := UntrustedQuery(urlFilter, urlVars...).Whitelist("Name", "Email")

	users := []User{}
	_ = db.Filter(context.Background(), &users, query)
	fmt.Printf("%+v", users)
	// Output:
	// INSERT INTO "user" ("id", "name", "email") VALUES ($1, $2, $3) ON CONFLICT ("id") DO UPDATE SET ("name", "email") = ROW("excluded"."name", "excluded"."email") RETURNING "id", "name", "email"
	// SELECT "id", "name", "email" FROM "user" WHERE "name" = $1 and "email" = $2 LIMIT 10
	// [{Id:user_6 Name:Karl Email:karl@company.co}]
}

func stringSliceToInterfaceSlice(in []string) []interface{} {
	out := make([]interface{}, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = in[i]
	}
	return out
}
