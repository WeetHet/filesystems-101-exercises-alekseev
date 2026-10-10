package server

import (
	"net/http"
	"path"
	"time"

	"github.com/gorilla/mux"

	"tests-runner/db"
	"tests-runner/exercises"
	"tests-runner/git"
	"tests-runner/util/errors"
	"tests-runner/util/iam"
	"tests-runner/util/sqldb"
	"tests-runner/util/ulid"
	"tests-runner/util/util/errutil"
)

func (s *Server) handleSubmitSolution(r *http.Request, acl *iam.ACL, params any) (result any, err error) {
	ctx := r.Context()
	log := s.log

	ghUserName := mux.Vars(r)["gh_user"]
	exerciseId := mux.Vars(r)["exercise"]
	hc := s.isHorsConcours(r)
	log.Infof(ctx, "submit: user = %q, exercise = %q, hc = %t", ghUserName, exerciseId, hc)

	repoUrl := s.userName2RepoUrl(ghUserName)
	repoDir := path.Join(s.reposDir, ghUserName)
	s.lockRepo(repoUrl)
	defer s.unlockRepo(repoUrl)

	// 0. Some sanity checks.

	_, ok := exercises.Exercises[exerciseId]
	if !ok {
		return nil, errors.ErrNotFound.WithMessagef("no such exercise: %q", exerciseId)
	}

	deadline, err := s.db.GetExerciseDeadline(ghUserName, exerciseId)
	if errors.Is(err, errors.ErrNotFound) {
		return nil, errors.ErrNotFound.WithMessagef("the problem is not available yet: %q", exerciseId)
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(deadline) && !hc {
		return nil, errors.ErrGone.WithMessagef("the deadline was %s", deadline.UTC().Format(time.RFC1123))
	}

	// 1. Checkout a solution.

	repo, err := s.db.GetRepoByURL(repoUrl)
	if err != nil && !sqldb.IsRecordNotFoundError(err) {
		return nil, err
	}

	if sqldb.IsRecordNotFoundError(err) {
		err = errutil.First(
			git.Clone(s.reposDir, repoUrl, ghUserName),
			git.CheckoutBranch(repoDir, "solutions"),
		)
		if err != nil {
			return nil, errors.ErrNotFound.WithReason(err)
		}
		log.Infof(ctx, "cloned repo %q", repoUrl)

		repo = &db.Repo{
			Id:  ulid.New(),
			Url: repoUrl,
		}
		if err = s.db.InsertRepo(repo); err != nil {
			return nil, err
		}
	}

	err = errutil.First(
		git.CheckoutBranch(repoDir, "solutions"),
		git.Clean(repoDir),
		git.Pull(repoDir),
	)
	if err != nil {
		return nil, errors.ErrInternal.WithReason(err)
	}

	log = log.WithField("repo", repo.Id)

	// 2. Register it in the database.

	commitId, err := git.GetRefHash(repoDir, "HEAD")
	if err != nil {
		return nil, errors.ErrInternal.WithReason(err)
	}
	log.Infof(ctx, "updated the repo to commit %q", commitId)

	submission := &db.Submission{
		Id:         ulid.New(),
		ExerciseId: exerciseId,
		RepoId:     repo.Id,
		CommitId:   commitId,
		HC:         hc,
		Requestor:  db.RequestorStudent,
	}
	if err = s.db.InsertSubmission(submission); err != nil {
		return nil, err
	}
	log.Infof(ctx, "submission is %q", submission.Id.String())

	// 3. Ready to go.

	s.vq.Submit(repo, submission)

	return submission, nil
}

func (s *Server) handleReverifySolutions(r *http.Request, acl *iam.ACL, params any) (result any, err error) {
	ctx := r.Context()
	log := s.log

	exerciseId := mux.Vars(r)["exercise"]
	log.Infof(ctx, "reverify: exercise = %q", exerciseId)

	newSubmissions := make([]*db.Submission, 0)
	err = s.db.InTx(ctx, func(tx *db.DB) error {
		submissions, err := s.db.ListLastSubmissions(exerciseId)
		if err != nil {
			return err
		}

		for _, s := range submissions {
			newSubmission := &db.Submission{
				Id:         ulid.New(),
				ExerciseId: s.ExerciseId,
				RepoId:     s.RepoId,
				CommitId:   s.CommitId,
				Requestor:  db.RequestorAlex,
			}
			if err := tx.InsertSubmission(newSubmission); err != nil {
				return err
			}

			log.Infof(ctx, "reverify: %s -> %s", s.Id, newSubmission.Id)
			newSubmissions = append(newSubmissions, newSubmission)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, x := range newSubmissions {
		repo, err := s.db.GetRepoById(x.RepoId)
		if err != nil {
			return nil, err
		}

		s.vq.Submit(repo, x)
	}

	return newSubmissions, nil
}
