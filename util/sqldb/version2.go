package sqldb

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"tests-runner/util/logger"
	"tests-runner/util/errors"
	"tests-runner/util/util/errutil"
)

// DbVersion is table, which is used to stored schema id and version
type DbVersion struct {
	ID  string `gorm:"primaryKey"`
	Ver int
}

// MigrateF2 is func for manual migrate Schema using GORMv2
type MigrateF2 func(ctx context.Context, orm *gorm.DB) (err error)

// Flag allows to customize migration behaviour
type Flag int

const (
	// FlagWithoutTX indicates that the particular migration should be applied outside of a transaction.
	// Such migrations should be written in a way that allows them to be re-applied, as it's possible for the
	// migration to fail before the new version is recorded in the DB.
	FlagWithoutTX Flag = 1 << iota
)

// Version2 descibes migration from version <=Ver to version Ver+1 using the Migrate function.
type Version2 struct {
	Ver     int
	Migrate MigrateF2
	Flags   Flag
}

func applyVersion2(ctx context.Context, orm *gorm.DB, id string, version Version2) (err error) {
	if version.Flags&FlagWithoutTX == 0 {
		return applyVersion2TX(ctx, orm, id, version)
	}

	return applyVersion2WithoutTX(ctx, orm, id, version)
}

func applyVersion2TX(ctx context.Context, orm *gorm.DB, id string, version Version2) (err error) {
	err = orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = version.Migrate(ctx, tx); err != nil {
			return errors.Wrapf(err, "Version(%d)", version.Ver)
		}
		if err = versionAdvance2(tx, id, version.Ver); err != nil {
			return errors.Wrapf(err, "Version(%d): set", version.Ver)
		}
		return nil
	})
	return errors.Wrap(err, "gorm.Transaction")
}

func applyVersion2WithoutTX(ctx context.Context, orm *gorm.DB, id string, version Version2) (err error) {
	if err = version.Migrate(ctx, orm); err != nil {
		return errors.Wrapf(err, "Version(%d)", version.Ver)
	}
	if err = versionAdvance2(orm, id, version.Ver); err != nil {
		return errors.Wrapf(err, "Version(%d): set", version.Ver)
	}

	return nil
}

// ManualMigrate2 state from current state version to high version using GORMv2. The provided `versions` slice is sorted
// using `Ver` in increasing order before applying the migrations.
func ManualMigrate2(ctx context.Context, orm *gorm.DB, id string, versions []Version2, opts ...MigrationOption) (err error) {
	errutil.BugOn(len(versions) == 0, "there must be >= 1 migration")

	defer func() { err = errors.Wrapf(err, "sqldb.ManualMigrate2(%s)", id) }()

	if id == "" {
		return errors.New("migrate: service ID is not specified")
	}

	if err = orm.AutoMigrate(&DbVersion{}); err != nil {
		return errors.Wrap(err, "gorm.AutoMigrate")
	}

	var migrationOpts migrationOptions
	for _, opt := range opts {
		opt.apply(&migrationOpts)
	}

	log := migrationOpts.logger
	if log == nil {
		log = logger.NewNoop()
	}

	ver, err := VersionGet2(ctx, orm, id)
	if IsRecordNotFoundError(err) {
		log.Infof(ctx, "empty database %s, performing all %d migration(s)", id, len(versions))
		err = nil
	}
	if err != nil {
		return err
	}

	for i := 1; i < len(versions); i++ {
		prev, cur := versions[i-1].Ver, versions[i].Ver
		errutil.BugOn(cur != prev+1, "db versions must be monotonic and contiguous, but got %d -> %d", prev, cur)
	}

	lastVer := versions[len(versions)-1].Ver
	if ver == lastVer+1 {
		log.Infof(ctx, "The DB is up-to-date: %s:%d.", id, ver)
		return nil
	}
	if ver > lastVer+1 {
		log.Errorf(ctx, "The DB is newer than the service: %s:%d.", id, ver)
		return fmt.Errorf("the db is too new")
	}
	log.Infof(ctx, "performing DB migration for %s:%d", id, ver)

	lastApplied := ver - 1
	for _, u := range versions {
		if u.Ver <= lastApplied {
			continue
		}

		if err = applyVersion2(ctx, orm, id, u); err != nil {
			log.Errorf(ctx, "failed to apply migration %s:%d", id, u.Ver)
			return err
		}
		lastApplied = u.Ver
	}
	if lastApplied >= ver {
		log.Infof(ctx, "migrations %d..%d for DB %s finished", ver, lastApplied, id)
	}

	return nil
}

// VersionGet returns current version using GORMv2
func VersionGet2(ctx context.Context, orm *gorm.DB, id string) (ver int, err error) {
	v := DbVersion{}
	err = orm.WithContext(ctx).First(&v, "id = ?", id).Error
	if err != nil {
		return 0, errors.Wrap(err, "gorm.Find")
	}

	return v.Ver, nil
}

func versionAdvance2(orm *gorm.DB, id string, ver int) (err error) {
	q := orm.Model(&DbVersion{}).
		Where("id = ?", id).
		Where("ver = ?", ver).
		Update("ver", gorm.Expr("ver + 1"))
	if q.Error != nil {
		return errors.Wrap(q.Error, "gorm.Update")
	}

	errutil.BugOn(q.RowsAffected > 1, "the db is broken")
	if q.RowsAffected == 1 {
		return nil
	}

	q = orm.Create(&DbVersion{ID: id, Ver: ver + 1})
	if IsConflictError(q.Error) {
		return fmt.Errorf("db upgrade raced with another pod")
	}
	return errors.Wrap(q.Error, "gorm.Create")
}

type migrationOptions struct {
	logger logger.Logger
}

// MigrationOption allows to customize migration process.
type MigrationOption interface {
	apply(*migrationOptions)
}

type migrationOptionFunc func(*migrationOptions)

func (f migrationOptionFunc) apply(o *migrationOptions) {
	f(o)
}

// MigrateWithLogger sets logger to use during migration. If no logger is provided, migration process is performed w/o
// any output.
func MigrateWithLogger(l logger.Logger) MigrationOption {
	return migrationOptionFunc(func(o *migrationOptions) {
		o.logger = l
	})
}

// SimpleMigration returns a migration function that applies all passed SQL statements.
func SimpleMigration(stmts ...string) MigrateF2 {
	return func(ctx context.Context, orm *gorm.DB) error {
		for _, stmt := range stmts {
			if err := orm.WithContext(ctx).Exec(stmt).Error; err != nil {
				return errors.Wrap(err, "gorm.Exec")
			}
		}

		return nil
	}
}

// ExecStatements executes all passed SQL statements.
func ExecStatements(ctx context.Context, orm *gorm.DB, ss ...string) (err error) {
	return SimpleMigration(ss...)(ctx, orm)
}
