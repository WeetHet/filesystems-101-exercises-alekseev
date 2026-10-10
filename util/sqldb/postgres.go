//go:build postgres

package sqldb

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"tests-runner/util/errors"
	"tests-runner/util/util/errutil"
)

func init() {
	gDialects.add(dialect{
		Name:          dialectPostgres,
		AddIndex:      addIndexPostgresql,
		AddForeignKey: addForeignKeyPostgresql,
		NewDialector:  newPostgresDialector,
	})
}

func newPostgresDialector(f *ConnectionFlags) (res gorm.Dialector, err error) {
	defer func() { err = errors.Wrap(err, "newPostgresDialector()") }()

	dsn := postgresDSNBuild(f)

	driverName := ""
	if f.CredentialsFile != "" && f.Dialect == string(dialectPostgres) {
		driverName = "cloudsql-pgx" + dsn
		drivers := sql.Drivers() // sql drivers return sorted array of drivers
		i := sort.SearchStrings(drivers, driverName)
		if len(drivers) == 0 || i >= len(drivers) || drivers[i] != driverName {
			_, err = pgxv5.RegisterDriver(driverName, cloudsqlconn.WithCredentialsFile(f.CredentialsFile))
			if err != nil {
				return nil, err
			}
		}
	}

	p := postgres.New(postgres.Config{
		DSN:        dsn,
		DriverName: driverName,
	})

	res = &pgDialector{p, p.(gorm.SavePointerDialectorInterface)}
	return res, nil
}

func postgresDSNBuild(f *ConnectionFlags) string {
	sslmode := "disable"
	if f.SSLMode != "" {
		sslmode = f.SSLMode
	}

	// google cloud sql instance have specific pseudo host like aa:bb:cc[:port]
	if delim := strings.LastIndex(f.Host, ":"); delim > 0 {
		host := f.Host[:delim]
		port := f.Host[delim+1:]
		if _, err := strconv.Atoi(port); err == nil {
			return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s password=%s",
				host, port, f.User, f.Database, sslmode, f.Password)
		}
	}
	return fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s",
		f.Host, f.User, f.Database, sslmode, f.Password)
}

type pgDialector struct {
	gorm.Dialector
	gorm.SavePointerDialectorInterface
}

func (d *pgDialector) Initialize(db *gorm.DB) (err error) {
	if err = d.Dialector.Initialize(db); err != nil {
		return err
	}

	return nil
}

func (d *pgDialector) Translate(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errRecordNotFound
	}

	pgErr, ok := err.(interface{ SQLState() string })
	if !ok {
		return err
	}

	switch pgErr.SQLState() {
	// 23001 (restrict_violation), 23503 (foreign_key_violation)
	case "23001", "23503":
		err = errForeignKey.WithReason(err)
	case "23505":
		err = errConflict.WithReason(err)
	case "55P03":
		err = errLockTimeout.WithReason(err)
	case "40001":
		err = errSerialise.WithReason(err)
	case "57P03":
		err = errUnavailable.WithReason(err)
	case "22021":
		// character_not_in_repertoire
		// e.g. invalid byte sequence for encoding "UTF8": 0x00
		err = errInvalidInputCharacter.WithReason(err)
	}
	return err
}

func addIndexPostgresql(db *gorm.DB, opts IndexOptions) error {
	stmt := addIndexStmtPostgresql(opts)
	return db.Exec(stmt).Error
}

func addIndexStmtPostgresql(opts IndexOptions) string {
	b := strings.Builder{}

	if opts.Unique {
		b.WriteString("CREATE UNIQUE INDEX ")
	} else {
		b.WriteString("CREATE INDEX ")
	}

	if opts.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}

	if opts.Name != "" {
		b.WriteString(opts.Name)
		b.WriteString(" ")
	}

	b.WriteString("ON ")
	b.WriteString(opts.Table)
	b.WriteString(" ")

	if opts.Method != "" {
		b.WriteString("USING ")
		b.WriteString(opts.Method)
		b.WriteString(" ")
	}

	b.WriteString("(")
	for n, c := range opts.Columns {
		b.WriteString(c)
		if n != len(opts.Columns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(") ")

	if len(opts.ValueColumns) != 0 {
		b.WriteString("INCLUDE (")
		for n, c := range opts.ValueColumns {
			b.WriteString(c)
			if n != len(opts.ValueColumns)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString(") ")
	}

	return strings.TrimSuffix(b.String(), " ")
}

func addForeignKeyPostgresql(db *gorm.DB, opts foreignKeyOptions) (err error) {
	errutil.BugOn(opts.Name == "", "fk name not specified")

	if opts.IfNotExists {
		var count int64
		err = db.Raw("SELECT count(*) FROM pg_constraint WHERE conname = ?", opts.Name).Scan(&count).Error
		if err != nil || count > 0 {
			return err
		}
	}

	stmt := addForeignKeyStatementDefault(opts)
	return db.Exec(stmt).Error
}
