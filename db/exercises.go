package db

import (
	"sync"
	"time"

	"tests-runner/util/errors"
	"tests-runner/util/sqldb"
	"tests-runner/util/ulid"
	"tests-runner/util/workgroup"
)

type Deadline struct {
	ExerciseId string
	Deadline   time.Time
}

func (db *DB) GetExerciseDeadline(user, exerciseId string) (deadline time.Time, err error) {
	defer func() { err = errors.Wrapf(err, "GetExerciseDeadline(%s)", exerciseId) }()

	var d Deadline
	err = db.orm.
		Where("gh_user = ? AND exercise_id = ?", user, exerciseId).
		First(&d).Error
	if sqldb.IsRecordNotFoundError(err) {
		err = db.orm.Where("exercise_id = ?", exerciseId).First(&d).Error
	}
	if sqldb.IsRecordNotFoundError(err) {
		return time.Time{}, errors.ErrNotFound
	}
	if err != nil {
		return time.Time{}, err
	}

	return d.Deadline, nil
}

type ExerciseStats struct {
	RepoId     ulid.ULID
	RepoUrl    string
	ExerciseId string

	FirstSubmission         *Submission
	FirstAcceptedSubmission *Submission
	LastSubmission          *Submission
}

func (db *DB) GetExerciseStats(hc bool) (res []ExerciseStats, err error) {
	defer func() { err = errors.Wrap(err, "GetExerciseStats()") }()

	var groups []struct {
		RepoId     ulid.ULID
		RepoUrl    string
		ExerciseId string
	}

	err = db.orm.
		Raw(`
SELECT DISTINCT s.repo_id as repo_id, r.url as repo_url, s.exercise_id as exercise_id
FROM submissions s, repos r
WHERE r.id = s.repo_id
`).
		Scan(&groups).Error
	if err != nil {
		return nil, err
	}

	wg := workgroup.New(workgroup.Config{Concurrency: 8})
	lock := sync.Mutex{}
	for _, g_ := range groups {
		g := g_

		wg.Go(func() error {
			s := ExerciseStats{
				RepoId:     g.RepoId,
				RepoUrl:    g.RepoUrl,
				ExerciseId: g.ExerciseId,
			}

			if err := db.fillExerciseStats(&s, hc); err != nil {
				return err
			}

			if s.FirstSubmission != nil {
				lock.Lock()
				res = append(res, s)
				lock.Unlock()
			}

			return nil
		})
	}
	wg.Wait()
	if err = wg.Error(); err != nil {
		return nil, err
	}

	return res, nil
}

func (db *DB) fillExerciseStats(s *ExerciseStats, hc bool) (err error) {
	err = db.orm.Raw(`
SELECT * FROM submissions WHERE id = (
	SELECT min(id) FROM submissions
	WHERE repo_id = ? AND exercise_id = ? AND hc = ?
)`,
		s.RepoId, s.ExerciseId, hc).
		First(&s.FirstSubmission).Error
	if sqldb.IsRecordNotFoundError(err) {
		s.FirstSubmission = nil
		err = nil
	}
	if err != nil {
		return err
	}

	err = db.orm.Raw(`
SELECT * FROM submissions WHERE id = (
	SELECT min(id) FROM submissions
	WHERE repo_id = ? AND exercise_id = ? AND hc = ? AND accepted
)`,
		s.RepoId, s.ExerciseId, hc).
		First(&s.FirstAcceptedSubmission).Error
	if sqldb.IsRecordNotFoundError(err) {
		s.FirstAcceptedSubmission = nil
		err = nil
	}
	if err != nil {
		return err
	}

	err = db.orm.Raw(`
SELECT * FROM submissions WHERE id = (
	SELECT max(id) FROM submissions
	WHERE repo_id = ? AND exercise_id = ? AND hc = ?
)`,
		s.RepoId, s.ExerciseId, hc).
		First(&s.LastSubmission).Error
	if sqldb.IsRecordNotFoundError(err) {
		s.LastSubmission = nil
		err = nil
	}
	if err != nil {
		return err
	}

	return nil
}
