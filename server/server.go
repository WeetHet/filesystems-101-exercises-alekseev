package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"tests-runner/util/errors"
	"tests-runner/util/locks"
	"tests-runner/util/logger"
	csrv "tests-runner/util/server"

	"tests-runner/db"
)

type Config struct {
	DB        *db.DB
	Log       logger.Logger
	ReposHost string
	ReposDir  string
}

type Server struct {
	db  *db.DB
	log logger.Logger

	reposHost string
	reposDir  string
	repoLock  *locks.MappedLock[string]

	vq *VerificationQueue
}

func New(cfg Config) *Server {
	return &Server{
		db:  cfg.DB,
		log: cfg.Log,

		reposHost: cfg.ReposHost,
		reposDir:  cfg.ReposDir,
		repoLock:  locks.NewMappedLock[string](),
	}
}

func (s *Server) Start(ctx context.Context) (err error) {
	s.vq = s.newVerificationQueue()

	if err = s.vq.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() {
	s.vq.Stop()
}

func (s *Server) Wait() {
	s.vq.Wait()
}

func (s *Server) IsHealthy(ctx context.Context) (err error) {
	return nil
}

func (s *Server) IsReady(ctx context.Context) bool {
	return true
}

func (s *Server) AddRoutes(cs *csrv.Server) (err error) {
	routes := []csrv.Route{
		{Method: http.MethodPost, Path: "/submit/{gh_user}/{exercise}", Handler: s.handleSubmitSolution},
		{Method: http.MethodGet, Path: "/submissions/{gh_user}/{exercise}", Handler: s.handleListSubmissions},
		{Method: http.MethodGet, Path: "/results/{gh_user}/{exercise}/{s}", Handler: s.handleGetSubmissionResult},

		{Method: http.MethodPost, Path: "/reverify/{exercise}", Handler: s.handleReverifySolutions},
		{Method: http.MethodGet, Path: "/stats/{axis}", Handler: s.handleGetExerciseStats},
		{Method: http.MethodGet, Path: "/results", RawHandler: s.handleGetResultsTable},
	}
	for i := range routes {
		routes[i].Flags |= csrv.RouteWithoutPrefix
		routes[i].Flags |= csrv.RouteWithoutAuth
		routes[i].Flags |= csrv.RouteWithoutACL
	}

	cs.AddRoutes(routes)
	return nil
}

func (s *Server) lockRepo(repoUrl string) {
	s.repoLock.Lock(repoUrl)
}

func (s *Server) unlockRepo(repoUrl string) {
	s.repoLock.Unlock(repoUrl)
}

func (s *Server) userName2RepoUrl(ghUserName string) string {
	return fmt.Sprintf("git@%s:%s/filesystems-101-exercises.git",
		s.reposHost, ghUserName)
}

var repoUrlRegex = regexp.MustCompile("git@(.+):(.+)/(.+).git")

func (s *Server) repoUrl2UserName(repoUrl string) (string, error) {
	matches := repoUrlRegex.FindStringSubmatch(repoUrl)
	if len(matches) != 4 {
		return "", errors.ErrInvalidArg.WithMessagef("malformed repo url: %q", repoUrl)
	}
	return matches[2], nil
}

func (s *Server) isHorsConcours(r *http.Request) bool {
	return (r.Header.Get("X-HorsConcours") == "true")
}
