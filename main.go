package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/orsinium/chameleon/chameleon"
	"go.uber.org/zap"
)

func run(logger *zap.Logger) error {
	config := chameleon.NewConfig().Parse()
	repo := chameleon.Repository{Path: chameleon.Path(config.RepoPath)}
	s := chameleon.Server{
		Repository: repo,
	}

	err := s.Init()
	if err != nil {
		return fmt.Errorf("cannot init repos: %v", err)
	}
	defer s.Close()

	sch := gocron.NewScheduler(time.UTC)
	_, err = sch.Every(config.Pull).Do(func() {
		logger.Debug("pulling the repo")
		err := repo.Pull()
		if err != nil {
			logger.Error("cannot pull the repo", zap.Error(err))
		}
	})
	if err != nil {
		return fmt.Errorf("cannot schedule the task: %v", err)
	}

	logger.Info("listening", zap.String("addr", config.Address))
	return s.Serve(config.Address)
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logger.Sync() // nolint

	logger.Info("starting...")
	err = run(logger)
	if err != nil {
		logger.Error("fatal error", zap.Error(err))
		os.Exit(1)
	}
}
