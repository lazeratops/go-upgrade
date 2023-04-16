package pkg

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
	Dir      string
}

// Sync either clones or updates a GitHub repository
func (r *Repo) Sync(dir string, force bool) error {
	r.Dir = dir
	p := filepath.Join(dir, r.Name)
	repo, err := git.PlainOpen(p)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return r.clone()
		}
		return fmt.Errorf("failed to open repo: %w", err)
	}
	// Pull repo, check out branch with upgrade
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %w", err)
	}

	var mainBranch *plumbing.ReferenceName

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to get references: %v", err)
	}
	refs.ForEach(func(ref *plumbing.Reference) error {
		if mainBranch != nil {
			return nil
		}
		n := ref.Name()
		ns := ref.Name().String()
		if strings.Contains(ns, "main") {
			mainBranch = &n
			return nil
		}
		if strings.Contains(ns, "master") {
			mainBranch = &n
			return nil
		}
		return nil
	})
	if mainBranch == nil {
		return ErrNoMainBranch
	}
	mb := *mainBranch
	h, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get head: %w", err)
	}
	if h.Name() != mb {
		if err := w.Checkout(&git.CheckoutOptions{
			Branch: mb,
		}); err != nil {
			return fmt.Errorf("%v: %w", err, ErrNoMainBranch)
		}
	}
	if force {
		commit := plumbing.NewHash("HEAD")
		if err := w.Reset(&git.ResetOptions{
			Mode:   git.HardReset,
			Commit: commit,
		}); err != nil {
			return fmt.Errorf("failed to reset repo: %w", err)
		}
	}

	if err := w.Pull(&git.PullOptions{}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			fmt.Errorf("failed to pull repo: %w", err)
		}
	}
	return nil
}

func (r *Repo) clone() error {
	d := r.getLocalRepoPath()
	_, err := git.PlainClone(d, false, &git.CloneOptions{
		URL:      r.CloneURL,
		Progress: os.Stdout,
	})

	if err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}
	return nil
}

func (r *Repo) UpgradeDeps() error {
	if !r.isNpmPkg() {
		return nil
	}
	cmd := exec.Command("/bin/sh", "-c", "npm outdated | awk 'NR>1 {print $1\"@\"$4}' | xargs npm install")

	repoPath := r.getLocalRepoPath()
	cmd.Dir = repoPath

	o, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to update dependencies: %w (%s)", err, o)
	}
	return nil
}

func (r *Repo) getLocalRepoPath() string {
	return filepath.Join(r.Dir, r.Name)
}

func (r *Repo) isNpmPkg() bool {
	repoPath := r.getLocalRepoPath()
	pkgJson := filepath.Join(repoPath, "package.json")
	fi, err := os.Stat(pkgJson)
	if os.IsNotExist(err) {
		return false
	}
	return !fi.IsDir()
}
