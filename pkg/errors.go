package pkg

import "errors"

var (
	ErrFailedRepoFetch = errors.New("failed to fetch repos from GitHub")
	ErrNoMainBranch    = errors.New("failed to check out main branch")
)
