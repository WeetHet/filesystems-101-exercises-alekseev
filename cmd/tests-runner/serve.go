package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"tests-runner/util/logger"
	csrv "tests-runner/util/server"
	"tests-runner/util/sqldb"
	"github.com/spf13/cobra"

	"tests-runner/db"
	"tests-runner/server"
)

var serveFlags struct {
	logger logger.Flags
	sqldb  sqldb.Flags
	csrv   csrv.Flags

	reposHost string
	reposDir  string
}

var serveCmd = cobra.Command{
	Use:  "serve",
	RunE: serve,
}

func init() {
	f := serveCmd.Flags()

	csrv.InjectFlags(f, &serveFlags.csrv)
	sqldb.InjectFlags(f, &serveFlags.sqldb)
	logger.InjectFlags(f, &serveFlags.logger)

	f.StringVar(&serveFlags.reposHost, "repos-host", "github.com", "github host alias in .ssh/config")
	f.StringVar(&serveFlags.reposDir, "repos", "./repos", "a directory to keep solution git repos")

	rootCmd.AddCommand(&serveCmd)
}

func serve(cmd *cobra.Command, args []string) (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := logger.NewFromFlags(&serveFlags.logger)

	dbLog := log.WithField(logger.KeySubsystem, "db")
	gormLog := dbLog.WithField(logger.KeyComponent, "gorm")
	orm, err := sqldb.NewGORM2FromFlags(&serveFlags.sqldb, gormLog)
	if err != nil {
		return err
	}

	db, err := db.New(db.Config{Orm: orm, Log: dbLog})
	if err != nil {
		return err
	}

	srvLog := log.WithField(logger.KeySubsystem, "srv")
	runTestsSubserver := server.New(server.Config{
		DB:        db,
		Log:       srvLog,
		ReposHost: serveFlags.reposHost,
		ReposDir:  serveFlags.reposDir,
	})

	cfg := csrv.ConfigFromFlags(&serveFlags.csrv)
	cfg.Log = log
	cfg.Version = "0.0.1"

	srv, err := csrv.New(ctx, cfg, runTestsSubserver)
	if err != nil {
		return err
	}

	err = srv.ListenAndServe(ctx)
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	return err
}
