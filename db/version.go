package db

import (
	"context"
	"encoding/json"

	"tests-runner/util/errors"
	"tests-runner/util/sqldb"
	"gorm.io/gorm"
)

const dbVersionId = "tests-runner"

var migrations = []sqldb.Version2{
	{Ver: 1, Migrate: upgradeV1},
	{Ver: 2, Migrate: upgradeV2},
	{Ver: 3, Migrate: upgradeV3},
	{Ver: 4, Migrate: upgradeV4},
	{Ver: 5, Migrate: upgradeV5},
	{Ver: 6, Migrate: upgradeV6},
}

func (db *DB) setup() (err error) {
	defer func() { err = errors.Wrap(err, "setup()") }()

	err = sqldb.ManualMigrate2(context.Background(),
		db.orm,
		dbVersionId, migrations)
	if err != nil {
		return err
	}

	return nil
}

const createTableRepos = `
CREATE TABLE repos(
	id		TEXT	PRIMARY KEY,
	url		TEXT	NOT NULL UNIQUE
)`

const createTableSubmissions = `
CREATE TABLE submissions(
	id		TEXT	PRIMARY KEY,
	exercise_id	TEXT	NOT NULL,
	repo_id		TEXT	NOT NULL,
	commit_id	TEXT	NOT NULL,
	result		TEXT,

	FOREIGN KEY (repo_id) REFERENCES repos(id)
)`

func upgradeV1(ctx context.Context, db *gorm.DB) (err error) {
	return execStatements(db.WithContext(ctx),
		createTableRepos,
		createTableSubmissions,
	)
}

const addSubmissionStatus = `
ALTER TABLE submissions
	ADD COLUMN result_summary TEXT,
	ADD COLUMN accepted BOOLEAN
`

func upgradeV2(ctx context.Context, db *gorm.DB) (err error) {
	if err = execStatements(db, addSubmissionStatus); err != nil {
		return err
	}

	var submissions []Submission
	if err = db.Find(&submissions).Error; err != nil {
		return err
	}

	for _, s := range submissions {
		var summary struct {
			Result string `json:"result"`
		}
		if s.Result != "" {
			if err = json.Unmarshal([]byte(s.Result), &summary); err != nil {
				return err
			}
		}

		err = db.Model(s).Updates(map[string]any{
			"result_summary": summary.Result,
			"accepted":       summary.Result == "ok",
		}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

const addRequestor = `
ALTER TABLE submissions
	ADD COLUMN requestor TEXT NOT NULL DEFAULT 's'
`

func upgradeV3(ctx context.Context, db *gorm.DB) (err error) {
	return execStatements(db, addRequestor)
}

const createTableDeadlines = `
CREATE TABLE deadlines (
	exercise_id TEXT NOT NULL UNIQUE,
	deadline    TIMESTAMP NOT NULL
)`

func upgradeV4(ctx context.Context, db *gorm.DB) (err error) {
	return execStatements(db, createTableDeadlines)
}

const addPerUserDeadlines = `
ALTER TABLE deadlines
	ADD COLUMN gh_user TEXT
`

const dropUniqueExerciseIdInDeadlines = `
ALTER TABLE deadlines
	DROP CONSTRAINT deadlines_exercise_id_key
`

const addUniqueExerciseAndUserInDeadlines = `
CREATE UNIQUE INDEX deadlines_exercise_id_user_key ON deadlines (exercise_id, gh_user)
`

func upgradeV5(ctx context.Context, db *gorm.DB) (err error) {
	return execStatements(db,
		addPerUserDeadlines,
		dropUniqueExerciseIdInDeadlines,
		addUniqueExerciseAndUserInDeadlines,
	)
}

const addHc = `
ALTER TABLE submissions
	ADD COLUMN hc BOOL NOT NULL DEFAULT FALSE
`

func upgradeV6(ctx context.Context, db *gorm.DB) (err error) {
	return execStatements(db, addHc)
}

func execStatements(orm *gorm.DB, ss ...string) (err error) {
	defer func() { err = errors.Wrap(err, "execStatements()") }()

	for _, s := range ss {
		err = orm.Exec(s).Error
		if err != nil {
			return err
		}
	}
	return nil
}
