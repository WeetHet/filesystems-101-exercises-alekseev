package sqldb

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"tests-runner/util/errors"
	"tests-runner/util/logger"
	"tests-runner/util/util/errutil"
)

// NewGORM2FromFlags returns a new instance of GORM2 configured using provided flags. Logger *must* not be nil. Optional
// opts arguments allow to customize GORMv2 with options not worth putting into command line arguments.
func NewGORM2FromFlags(f *Flags, log logger.Logger, opts ...func(*gorm.Config)) (*gorm.DB, error) {
	errutil.BugOn(log == nil, "log must not be nil")

	dialector, err := gDialects.newDialector(&f.ConnectionFlags)
	if err != nil {
		return nil, err
	}

	cfg := &gorm.Config{
		Logger:  newGORM2Logger(f.LoggingFlags, log),
		NowFunc: time.Now,

		// By default every Transaction() called inside another Transaction()
		// causes a SAVEPOINT creation. But we don't use rollbacks to SAVEPOINTs
		// anywhere, so, lets disable SAVEPOINTs by default.
		DisableNestedTransaction: true,

		// By default, GORM wraps each CREATE, UPDATE and DELETE into
		// a transaction. That makes sense for ORM-like workloads where
		// a single call to Create() can insert data into multiple tables
		// (a record and records that it references).
		//
		// CRUD operations that update a single table are translated into
		// one SQL query because GORM can use RETURNING and ON CONFLICT.
		//
		// Single-table updates are the only scenario possible in our code
		// because we forbid using GORM as anything but a generator of
		// parametrised SELECTs, INSERTs, UPDATEs and DELETEs. In these
		// cases we can save RTTs to the DB.
		SkipDefaultTransaction: true,

		// Translate FK conflict and other errors from driver specific to *our* errors.
		// To do so, we overload Dialector with our own .Translate() method.
		// NOTE: std gorm drivers may also translate driver specific errors to GORM errors,
		//    however, these errors are not detailed :( e.g. it doesn't print name of constraint hit.
		TranslateError: true,

		// Use pgx's default rules about preparing statements and caching them.
		// GORM's Create() pollutes the cache of prepared statements. It can
		// be fixed in pgx's dialector. GORM's own cache stays broken, so keep
		// it disabled.
		//
		// See also pgDialector.pgBuildValuesClause().
		PrepareStmt: false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	db, err := gorm.Open(dialector, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "gorm.Open")
	}

	stdDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "db.DB")
	}
	configureStdDB(stdDB, &f.ConfigFlags)

	if !f.NoTracing {
		db = withSQLMonitoring(db, f.Database)
	}

	return db, nil
}

func configureStdDB(db *sql.DB, f *ConfigFlags) {
	db.SetMaxOpenConns(f.MaxOpenConnections)
	// SetMaxOpenConns(0) ==> unlimited open conns.
	// SetMaxIdleConns(0) ==> 0 idle conns.
	// ==> lets keep the default number of idle conns
	//     if 0 max open connections is specified.
	if f.MaxOpenConnections > 0 {
		db.SetMaxIdleConns(f.MaxOpenConnections)
	} else {
		// Test environments do not care to set DB_MAX_OPEN_CONNECTIONS.
		// Choose a reasonably sized idle pool so that most connections
		// can be reused.
		db.SetMaxIdleConns(64)
	}
	db.SetConnMaxLifetime(f.MaxConnectionLifetime)
}

type gorm2Logger struct {
	l      logger.Logger
	thresh time.Duration
}

func newGORM2Logger(f LoggingFlags, log logger.Logger) gormLogger.Interface {
	l := log.WithLevel(f.LogLevel)
	if !f.LogEnable {
		l = logger.NewNoop()
	}

	return &gorm2Logger{
		l:      l,
		thresh: f.LogSlowQueryThreshold,
	}
}

// LogMode implements GORMv2 logger.Interface
func (l *gorm2Logger) LogMode(lvl gormLogger.LogLevel) gormLogger.Interface {
	log := l.l
	switch lvl {
	case gormLogger.Error:
		log = log.WithLevel(logger.Error)
	case gormLogger.Warn:
		log = log.WithLevel(logger.Warning)
	case gormLogger.Info:
		log = log.WithLevel(logger.Info)
	case gormLogger.Silent:
		log = logger.NewNoop()
	}

	return &gorm2Logger{l: log, thresh: l.thresh}
}

// Info implements GORMv2 logger.Interface
func (l *gorm2Logger) Info(ctx context.Context, msg string, args ...any) {
	l.l.Infof(ctx, msg, args...)
}

// Warn implements GORMv2 logger.Interface
func (l *gorm2Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.l.Warningf(ctx, msg, args...)
}

// Error implements GORMv2 logger.Interface
func (l *gorm2Logger) Error(ctx context.Context, msg string, args ...any) {
	l.l.Errorf(ctx, msg, args...)
}

// Trace implements GORMv2 logger.Interface
func (l *gorm2Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	elapsedMS := float64(elapsed.Nanoseconds()) / 1e6

	switch {
	case err != nil && l.l.Enabled(logger.Error) && !errors.Is(err, errRecordNotFound):
		query, nrRows := fc()
		l.l.WithError(err).Errorf(ctx, "%s: SQL query error [%.3fms] [rows:%s] %s",
			utils.FileWithLineNum(), elapsedMS, formatNrRows(nrRows), query)
	case l.thresh != 0 && elapsed > l.thresh && l.l.Enabled(logger.Warning):
		query, nrRows := fc()
		l.l.Warningf(ctx, "%s: slow SQL query [%.3fms] [rows:%s] %s",
			utils.FileWithLineNum(), elapsedMS, formatNrRows(nrRows), query)
	case l.l.Enabled(logger.Debug):
		query, nrRows := fc()
		l.l.Debugf(ctx, "%s: SQL query [%.3fms] [rows: %s] %s",
			utils.FileWithLineNum(), elapsedMS, formatNrRows(nrRows), query)
	}
}

func formatNrRows(nrRows int64) string {
	if nrRows == -1 {
		return "-"
	} else {
		return strconv.FormatInt(nrRows, 10)
	}
}
