package main

import (
	"github.com/alecthomas/kong"
	"go-upgrade/pkg"
	"go.uber.org/zap"
	"os"
	"sync"
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
	cloner := pkg.NewOrgCloner(cli.GitHubOrg, pkg.WithToken(cli.GitHubToken), pkg.WithSyncDir(syncDir), pkg.WithForce(cli.Force))
	sugar.Infof("Syncing all repos in %s", cli.GitHubOrg)
	repos, err := cloner.SyncAllRepos()
	if err != nil {
		sugar.Fatalf("Failed to sync all repos: %v", err)
	}

	l := len(repos)
	sugar.Infof("Updated %d repositories in %s", l, cli.GitHubOrg)

	var wg sync.WaitGroup
	wg.Add(l)
	for _, r := range repos {
		r := r
		go func() {
			sugar.Infof("Updating %s", r.Name)
			if err := r.UpgradeDeps(); err != nil {
				sugar.Errorf("Failed to upgrade repo %s: %v", r.Name, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
