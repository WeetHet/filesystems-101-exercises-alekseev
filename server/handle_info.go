package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/cli-runtime/pkg/printers"

	"tests-runner/db"
	"tests-runner/util/container/set"
	"tests-runner/util/errors"
	"tests-runner/util/iam"
	"tests-runner/util/ulid"
)

type listedSubmission struct {
	Id          ulid.ULID `json:"id"`
	SubmittedAt string    `json:"submitted_at"`
	CommitId    string    `json:"commit_id"`
	HasResult   bool      `json:"has_result"`
}

func (s *Server) handleListSubmissions(r *http.Request, acl *iam.ACL, params any) (result any, err error) {
	ghUserName := mux.Vars(r)["gh_user"]
	exerciseId := mux.Vars(r)["exercise"]

	repoUrl := s.userName2RepoUrl(ghUserName)
	repo, err := s.db.GetRepoByURL(repoUrl)
	if err != nil {
		return nil, err
	}

	submissions, err := s.db.ListSubmissions(repo.Id, exerciseId)
	if err != nil {
		return nil, err
	}

	res := make([]listedSubmission, len(submissions))
	for i, s := range submissions {
		res[i] = listedSubmission{
			Id:          s.Id,
			SubmittedAt: s.Id.Time().Format(time.RFC822Z),
			CommitId:    s.CommitId,
			HasResult:   s.Result != "",
		}
	}
	return res, nil
}

func (s *Server) handleGetSubmissionResult(r *http.Request, acl *iam.ACL, params any) (result any, err error) {
	var submission *db.Submission

	ghUserName := mux.Vars(r)["gh_user"]
	exerciseId := mux.Vars(r)["exercise"]
	submissionIdStr := mux.Vars(r)["s"]

	repoUrl := s.userName2RepoUrl(ghUserName)
	repo, err := s.db.GetRepoByURL(repoUrl)
	if err != nil {
		return nil, err
	}

	switch submissionIdStr {
	case "last":
		submissions, err := s.db.ListSubmissions(repo.Id, exerciseId)
		if err != nil {
			return nil, err
		}
		if len(submissions) == 0 {
			return nil, errors.ErrNotFound.WithMessage("no submissions")
		}
		submission = &submissions[len(submissions)-1]

	default:
		submissionId, err := ulid.Parse(submissionIdStr)
		if err != nil {
			return nil, errors.ErrInvalidArg.WithMessagef("malformed submission id: %q", submissionIdStr)
		}

		submission, err = s.db.GetSubmissionById(submissionId)
		if err != nil {
			return nil, err
		}
	}

	if submission.Result == "" {
		return nil, errors.ErrNotFound.WithMessage("not verified yet")
	}

	return json.RawMessage(submission.Result), nil
}

type statE struct {
	Exercise string `json:"exercise"`

	Accepted bool `json:"accepted"`

	First         string `json:"first"`
	FirstAccepted string `json:"first_accepted,omitempty"`
	Last          string `json:"last,omitempty"`
}

type PerUserExerciseStats struct {
	User      string  `json:"user"`
	Exercises []statE `json:"exercises"`
}

type statU struct {
	User string `json:"user"`

	Accepted bool `json:"accepted"`

	First         string `json:"first"`
	FirstAccepted string `json:"first_accepted,omitempty"`
	Last          string `json:"last,omitempty"`
}

type PerExerciseUserStats struct {
	Exercise string  `json:"exercise"`
	Users    []statU `json:"users"`
}

func (s *Server) handleGetExerciseStats(r *http.Request, acl *iam.ACL, params any) (result any, err error) {
	axis := mux.Vars(r)["axis"]
	hc := s.isHorsConcours(r)

	stats, err := s.db.GetExerciseStats(hc)
	if err != nil {
		return nil, err
	}

	for i := range stats {
		x := &stats[i]
		user, err := s.repoUrl2UserName(x.RepoUrl)
		if err != nil {
			return nil, err
		}
		x.RepoUrl = user
	}

	switch axis {
	case "u":
		sort.Slice(stats, func(i, j int) bool {
			x, y := &stats[i], &stats[j]
			r := bytes.Compare(x.RepoId[:], y.RepoId[:])
			if r < 0 {
				return true
			} else if r > 0 {
				return false
			} else {
				return x.ExerciseId < y.ExerciseId
			}
		})

		res := make([]*PerUserExerciseStats, 0)
		var s *PerUserExerciseStats
		for _, x := range stats {
			if s == nil || s.User != x.RepoUrl {
				s = &PerUserExerciseStats{User: x.RepoUrl}
				res = append(res, s)
			}
			s.Exercises = append(s.Exercises, statE{
				Exercise:      x.ExerciseId,
				Accepted:      (x.LastSubmission != nil) && x.LastSubmission.Accepted,
				First:         submissionTs(x.FirstSubmission),
				FirstAccepted: submissionTs(x.FirstAcceptedSubmission),
				Last:          submissionTs(x.LastSubmission),
			})
		}
		return res, nil

	case "e":
		sort.Slice(stats, func(i, j int) bool {
			x, y := &stats[i], &stats[j]
			if x.ExerciseId < y.ExerciseId {
				return true
			} else if x.ExerciseId > y.ExerciseId {
				return false
			} else {
				return bytes.Compare(x.RepoId[:], y.RepoId[:]) < 0
			}
		})

		res := make([]*PerExerciseUserStats, 0)
		var s *PerExerciseUserStats
		for _, x := range stats {
			if s == nil || s.Exercise != x.ExerciseId {
				s = &PerExerciseUserStats{Exercise: x.ExerciseId}
				res = append(res, s)
			}
			s.Users = append(s.Users, statU{
				User:          x.RepoUrl,
				Accepted:      (x.LastSubmission != nil) && x.LastSubmission.Accepted,
				First:         submissionTs(x.FirstSubmission),
				FirstAccepted: submissionTs(x.FirstAcceptedSubmission),
				Last:          submissionTs(x.LastSubmission),
			})
		}
		return res, nil

	default:
		return nil, errors.ErrInvalidArg.WithMessagef("invalid axis: %q", axis)
	}
}

func (s *Server) handleGetResultsTable(w http.ResponseWriter, r *http.Request, acl *iam.ACL, params any) (err error) {
	hc := s.isHorsConcours(r)

	stats, err := s.db.GetExerciseStats(hc)
	if err != nil {
		return err
	}

	for i := range stats {
		x := &stats[i]
		user, err := s.repoUrl2UserName(x.RepoUrl)
		if err != nil {
			return err
		}
		x.RepoUrl = user
	}

	statsM := make(map[string]map[string]db.ExerciseStats)
	usersS := set.NewStrSet()
	exercisesS := set.NewStrSet()
	for _, x := range stats {
		usersS.Add(x.RepoUrl)
		exercisesS.Add(x.ExerciseId)

		byU, haveU := statsM[x.RepoUrl]
		if !haveU {
			byU = make(map[string]db.ExerciseStats)
			statsM[x.RepoUrl] = byU
		}

		byU[x.ExerciseId] = x
	}

	users := usersS.AsSlice()
	sort.Slice(users, func(i, j int) bool {
		nrSolvedI := 0
		for _, x := range statsM[users[i]] {
			if x.LastSubmission.Accepted {
				nrSolvedI++
			}
		}

		nrSolvedJ := 0
		for _, x := range statsM[users[j]] {
			if x.LastSubmission.Accepted {
				nrSolvedJ++
			}
		}

		if nrSolvedI > nrSolvedJ {
			return true
		} else if nrSolvedI < nrSolvedJ {
			return false
		} else {
			return users[i] < users[j]
		}
	})

	exercises := exercisesS.AsSlice()
	sort.Strings(exercises)

	p := printers.GetNewTabWriter(w)
	fmt.Fprintf(p, "repo")
	for _, e := range exercises {
		fmt.Fprintf(p, "\t%s", e[0:2])
	}
	fmt.Fprintf(p, "\n")

	for _, u := range users {
		fmt.Fprintf(p, "%s", u)
		for _, e := range exercises {
			switch result, submitted := statsM[u][e]; {
			case submitted && result.LastSubmission.Accepted:
				fmt.Fprintf(p, "\t+")
			case submitted && !result.LastSubmission.Accepted:
				fmt.Fprintf(p, "\t-")
			case !submitted:
				fmt.Fprintf(p, "\t.")
			}
		}
		fmt.Fprintf(p, "\n")
	}
	p.Flush()

	return nil
}

func submissionTs(s *db.Submission) string {
	if s == nil {
		return ""
	} else {
		return s.Id.Time().Format(time.RFC1123)
	}
}
