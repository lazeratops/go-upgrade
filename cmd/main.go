package main

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"go-upgrade/pkg/git"
	"go-upgrade/pkg/upgrade"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"os"
	"sync"
	"time"
)

var cli struct {
	GitHubToken string   `short:"t" help:"GitHub token" type:"string" env:"GITHUB_TOKEN"`
	GitHubOrg   string   `short:"o" help:"GitHub organization" type:"string" env:"GITHUB_ORG" required:""`
	SyncDir     string   `short:"d" help:"Directory to sync to" type:"path" env:"SYNC_DIR" required:""`
	Repos       []string `short:"r" help:"Repos to sync"`
	Force       bool     `short:"f" help:"Force even if repos already have unstaged changes" type:"bool"`
}

func main() {
	kong.Parse(&cli)
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	defer logger.Sync()

	syncDir := cli.SyncDir
	// Create sync dir if it does not already exist
	fi, err := os.Stat(syncDir)
	if os.IsNotExist(err) || !fi.IsDir() {
		if err := os.Mkdir(syncDir, os.ModePerm); err != nil {
			sugar.Fatalf("Failed to create sync directory: %v", err)
		}
	}

	bn := fmt.Sprintf("autoupgrade-%s", time.Now().Format("2006-01-02-15-04-05"))
	cloner := git.NewOrgCloner(cli.GitHubOrg,
		git.WithToken(cli.GitHubToken),
		git.WithSyncDir(syncDir),
		git.WithForce(cli.Force),
		git.WithWorkingBranchName(bn))
	sugar.Infof("Syncing all repos in %s", cli.GitHubOrg)

	var syncedRepos []*git.Repo
	var repoSyncErr error

	rts := cli.Repos
	if rts == nil || len(rts) == 0 {
		syncedRepos, repoSyncErr = cloner.SyncAllRepos()
	} else {
		syncedRepos, repoSyncErr = cloner.SyncRepos(rts)
	}
	if repoSyncErr != nil {
		sugar.Fatalf("Failed to sync repos: %v", err)
	}

	l := len(syncedRepos)
	sugar.Infof("Updated %d repositories in %s", l, cli.GitHubOrg)

	var wg sync.WaitGroup
	wg.Add(l)

	// We'll run up to 10 concurrent upgrades here
	sem := semaphore.NewWeighted(5)
	ctx := context.TODO()
	for _, r := range syncedRepos {
		sem.Acquire(ctx, 1)
		r := r
		go func() {
			defer sem.Release(1)
			defer wg.Done()
			sugar.Infof("Updating %s", r.Name)
			if err := upgrade.Upgrade(r.LocalRepoPath()); err != nil {
				sugar.Errorf("Failed to upgrade repo %s: %v", r.Name, err)
			}
		}()
	}
	wg.Wait()
}
