package db

import (
	"tests-runner/util/errors"
	"tests-runner/util/ulid"
)

type Requestor string

const (
	RequestorStudent = Requestor("s")
	RequestorAlex    = Requestor("a")
)

type Submission struct {
	Id            ulid.ULID `json:"id"`
	ExerciseId    string    `json:"exercise_id"`
	RepoId        ulid.ULID `json:"repo_id"`
	CommitId      string    `json:"commit_id"`
	HC            bool      `json:"-"`
	Requestor     Requestor `json:"-"`
	Result        string    `json:"result,omitempty"`
	ResultSummary string    `json:"-"`
	Accepted      bool      `json:"-"`
}

func (db *DB) GetSubmissionById(id ulid.ULID) (s *Submission, err error) {
	defer func() { err = errors.Wrapf(err, "GetSubmissionById(%s)", id) }()

	s = &Submission{}
	if err = db.orm.Where("id = ?", id).First(s).Error; err != nil {
		return nil, err
	}

	return s, nil
}

func (db *DB) InsertSubmission(s *Submission) (err error) {
	defer func() { err = errors.Wrapf(err, "InsertSubmission(%q)", s.CommitId) }()

	return db.orm.Create(s).Error
}

func (db *DB) SetSubmissionResult(s *Submission, result, resultSummary string, accepted bool) (err error) {
	defer func() { err = errors.Wrap(err, "SetSubmissionResult()") }()

	return db.orm.Model(s).Updates(map[string]any{
		"result":         result,
		"result_summary": resultSummary,
		"accepted":       accepted,
	}).Error
}

func (db *DB) ListSubmissions(repoId ulid.ULID, exerciseId string) (res []Submission, err error) {
	defer func() { err = errors.Wrap(err, "ListSubmissions()") }()

	err = db.orm.
		Where("repo_id = ? AND exercise_id = ?", repoId, exerciseId).
		Order("id").
		Find(&res).
		Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (db *DB) ListLastSubmissions(exerciseId string) (res []Submission, err error) {
	defer func() { err = errors.Wrap(err, "ListLastSubmissions()") }()

	err = db.orm.Raw(`
SELECT * FROM submissions WHERE id IN (
	SELECT max(id) FROM submissions
	WHERE exercise_id = ?
	GROUP BY repo_id
)`,
		exerciseId).
		Scan(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}
