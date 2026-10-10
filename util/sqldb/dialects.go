package sqldb

import (
	"fmt"
	"sort"

	"gorm.io/gorm"

	"tests-runner/util/util/errutil"
)

// dialectName is a special "enum" type for dialects. All dialects
// must have very specific names to match gorm.DB.Dialector.Name()
type dialectName string

// dialect constants are listed here for information.
const (
	dialectPostgres = dialectName("postgres")
	dialectSqlite   = dialectName("sqlite")
	dialectMysql    = dialectName("mysql")
)

// gDialects is a global dialect registry used to build GORM
// dialectors from connection flags.
var gDialects = dialectRegistry{dialects: map[dialectName]dialect{}}

type dialect struct {
	// Name is a proper dialector name, e.g. 'sqlite', 'postgres', etc.
	Name dialectName
	// AddForeignKey can be implemented in order to support FK addition
	// using sqldb.AddForeignKey function. If not implemented then FK will
	// not be supported and sqldb.AddForeignKey function will be NOOP.
	AddForeignKey func(*gorm.DB, foreignKeyOptions) error
	// AddIndex can be implemented in order to support Indices addition
	// using sqldb.AddIndex/AddIndexOpts functions. If not implemented
	// then these functions will return error.
	AddIndex func(*gorm.DB, IndexOptions) error
	// NewDialector should return GORM dialector which properly
	// converts errors from the driver "format" to our pkg errors.
	NewDialector func(*ConnectionFlags) (gorm.Dialector, error)
}

type dialectRegistry struct {
	dialects map[dialectName]dialect
}

func (r *dialectRegistry) getRegisteredDialectsNames() (names []string) {
	for name := range r.dialects {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}

func (r *dialectRegistry) get(name string) (dialect, error) {
	if d, ok := r.dialects[dialectName(name)]; ok {
		return d, nil
	}
	return dialect{}, ErrUnsupportedDialect.WithMessagef("sqldb: get: unsupported dialect: %q", name)
}

func (r *dialectRegistry) add(d dialect) {
	errutil.BugOn(d.Name == "", "empty dialect name")
	errutil.BugOn(d.NewDialector == nil, "NewDialector not specified")
	if _, err := r.get(string(d.Name)); err == nil {
		panic(fmt.Sprintf("dialect already registered: %q", d.Name))
	}
	r.dialects[d.Name] = d
}

func (r *dialectRegistry) newDialector(f *ConnectionFlags) (gorm.Dialector, error) {
	dialect, err := r.get(f.Dialect)
	if err != nil {
		return nil, err
	}

	return dialect.NewDialector(f)
}
