package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestNewTransaction_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestNewTransaction(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestNewTransaction_Struct{})))

	// insert new record in successful transaction
	{
		tx, err := db.NewTransaction()
		require.NoError(t, err)
		require.NoError(t, tx.Save(context.Background(), &TestNewTransaction_Struct{"foo", "bar"}))
		require.NoError(t, tx.Commit())
	}

	// cause unique_violation error and rollback transaction
	{
		tx, err := db.NewTransaction()
		require.NoError(t, err)

		_, err = tx.Exec(context.Background(), "INSERT INTO test_new_transaction_struct (col1) VALUES ('foo')")
		requirePQError(t, err, "unique_violation")

		_, err = tx.Exec(context.Background(), "INSERT INTO test_new_transaction_struct (col1) VALUES ('foo')")
		requirePQError(t, err, "in_failed_sql_transaction")

		require.NoError(t, tx.Rollback()) // works
		require.Error(t, tx.Commit())     // this will fail at this point
	}

	// cause unique_violation error and try to commit failed transaction
	{
		tx, err := db.NewTransaction()
		require.NoError(t, err)

		_, err = tx.Exec(context.Background(), "INSERT INTO test_new_transaction_struct (col1) VALUES ('foo')")
		requirePQError(t, err, "unique_violation")

		_, err = tx.Exec(context.Background(), "INSERT INTO test_new_transaction_struct (col1) VALUES ('foo')")
		requirePQError(t, err, "in_failed_sql_transaction")

		require.Error(t, tx.Commit())   // fails, because transaction failed
		require.Error(t, tx.Rollback()) // this will fail at this point
	}
}

type TestTransaction_Struct struct {
	Col1 string `db:"pk"`
	Col2 string
}

func TestTransaction(t *testing.T) {
	db, err := Open(postgresURI)
	require.NoError(t, err)

	// attach test logger
	log := &testLogger{}
	db.Logger = log

	// create table
	require.NoError(t, db.ensureTable(mustNewMetaStruct(&TestTransaction_Struct{})))

	// insert new record in successful transaction
	{
		err := db.Transaction(func(tx *Transaction) error {
			if err := tx.Save(context.Background(), &TestTransaction_Struct{"foo", "bar"}); err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
	}

	// cause unique_violation error and rollback transaction
	{
		var rescueTx *Transaction
		err := db.Transaction(func(tx *Transaction) error {
			rescueTx = tx

			_, err := tx.Exec(context.Background(), "INSERT INTO test_transaction_struct (col1) VALUES ('foo')")
			return err
		})
		requirePQError(t, err, "unique_violation")
		require.Error(t, rescueTx.Commit()) // transaction has already been rolled back
	}
}
