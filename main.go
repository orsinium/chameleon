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
	var err error
	config := chameleon.NewConfig().Parse()
	repo := chameleon.Repository{Path: chameleon.Path(config.RepoPath)}

	cache, err := chameleon.NewCache(config.Cache)
	if err != nil {
		return fmt.Errorf("cannot create cache: %v", err)
	}

	server := chameleon.Server{
		Repository: repo,
		Database:   &chameleon.Database{},
		Logger:     logger,
		Cache:      cache,
		Auth: chameleon.Auth{
			Password: config.Password,
			Logger:   logger,
		},
	}

	err = server.Database.Open(config.DBPath)
	if err != nil {
		return fmt.Errorf("cannot open database: %v", err)
	}

	err = repo.Clone(config.RepoURL)
	if err != nil {
		return fmt.Errorf("cannot clone repo: %v", err)
	}

	server.Init(config.PProf)
	defer func() {
		err := server.Close()
		if err != nil {
			logger.Error("cannot close connection", zap.Error(err))
		}
	}()

	if config.Pull != 0 {
		sch := gocron.NewScheduler(time.UTC)
		job, err := sch.Every(config.Pull).Do(func() {
			logger.Debug("pulling the repo")
			err := repo.Pull()
			if err != nil {
				logger.Error("cannot pull the repo", zap.Error(err))
			}
		})
		if err != nil {
			return fmt.Errorf("cannot schedule the task: %v", err)
		}
		job.SingletonMode()
		sch.StartAsync()
	}

	logger.Info("listening", zap.String("addr", config.Address))
	return server.Serve(config.Address)
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
