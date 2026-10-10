package sqldb

import (
	"strings"

	"gorm.io/gorm"

	"tests-runner/util/errors"
)

type IndexOptions struct {
	Table string

	// Postgresql supports various indexing methods like "btree", "hash", etc.
	Method string

	Unique      bool
	IfNotExists bool

	// Index name. If empty, a default name will be generated.
	// Warning: in some databases you cannot use default names if two indices
	// are built over the same list of fields.
	Name string

	// The list of indexed columns.
	Columns []string
	// Postgresql supports storing non-indexed columns in the index. Such columns
	// can be accessed instantly when doing an index lookup, but increase the index
	// size.
	ValueColumns []string
}

// AddIndex adds an index if the DB supports it
func AddIndex(db *gorm.DB, table, method string, columns ...string) (err error) {
	return AddIndexOpts(db, IndexOptions{
		Table:   table,
		Method:  method,
		Columns: columns,
	})
}

// AddIndexOpts adds an index of the DB supports it
func AddIndexOpts(db *gorm.DB, opts IndexOptions) (err error) {
	dialect, err := gDialects.get(db.Dialector.Name())
	if err != nil {
		return err
	}
	if dialect.AddIndex == nil {
		return errors.Errorf("sqldb: AddIndex not implemented for dialect %q", dialect.Name)
	}
	return dialect.AddIndex(db, opts)
}

func genIdxName(opts IndexOptions) string {
	parts := make([]string, 0, len(opts.Columns)+2)
	parts = append(parts, opts.Table)
	parts = append(parts, opts.Columns...)
	parts = append(parts, "idx")
	return strings.Join(parts, "_")
}
