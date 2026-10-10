package sqldb

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"tests-runner/util/util/errutil"
)

// TxParams defines properties of a DB transaction to run.
type TxParams struct {
	IsolationLevel sql.IsolationLevel

	// Allow only reads in the transaction.
	RO bool

	// Verify all UNIQUE and FOREIGN KEY constraints, and constraint triggers
	// at the end of the transaction.
	//
	// NOTE: constraints and triggers need to be declared DEFERRABLE for
	// this setting to take effect:
	//     CREATE TABLE example (
	//         ...
	//         ref TEXT NOT NULL REFERENCES foo(id) DEFERRABLE
	//     )
	DeferConstraints bool
}

// InTx runs a function do in a DB transaction.
func InTx(ctx context.Context, db *gorm.DB, params TxParams, do func(ctx context.Context, tx *gorm.DB) error) (err error) {
	if params.IsolationLevel == sql.LevelDefault {
		params.IsolationLevel = sql.LevelReadCommitted
	}
	errutil.BugOn(params.IsolationLevel != sql.LevelReadCommitted &&
		params.IsolationLevel != sql.LevelSerializable,
		"transaction isolation level too fancy: %s", params.IsolationLevel)

	opts := sql.TxOptions{
		Isolation: params.IsolationLevel,
		ReadOnly:  params.RO,
	}

	f := func(tx *gorm.DB) error {
		if params.DeferConstraints {
			err := tx.WithContext(ctx).
				Exec("SET CONSTRAINTS ALL DEFERRED").
				Error
			if err != nil {
				return err
			}
		}

		return do(ctx, tx)
	}

	return db.WithContext(ctx).Transaction(f, &opts)
}
