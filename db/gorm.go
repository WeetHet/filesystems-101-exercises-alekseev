package db

import (
	"context"

	"tests-runner/util/logger"
	"tests-runner/util/util/errutil"
	"gorm.io/gorm"
)

type Config struct {
	Orm *gorm.DB
	Log logger.Logger
}

func (cfg *Config) validateAndComplete() {
	errutil.BugOn(cfg.Orm == nil, "db: nil Orm in config")
	if cfg.Log == nil {
		cfg.Log = logger.NewNoop()
	}
}

type DB struct {
	orm *gorm.DB
	log logger.Logger
}

func New(cfg Config) (*DB, error) {
	cfg.validateAndComplete()

	db := &DB{
		orm: cfg.Orm,
		log: cfg.Log,
	}

	if err := db.setup(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) clone() *DB {
	c := *db
	return &c
}

func (db *DB) InTx(ctx context.Context, fn func(db *DB) error) (err error) {
	return db.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDB := db.clone()
		txDB.orm = tx
		return fn(txDB)
	})
}
