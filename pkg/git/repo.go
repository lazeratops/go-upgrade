package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"path/filepath"
	"strings"
)

type Repo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	CloneURL string `json:"ssh_url"`
	Dir      string
}

// Sync either clones or updates a GitHub repository
func (r *Repo) Sync(dir string, clean bool, branchName string) error {
	r.Dir = dir
	p := filepath.Join(dir, r.Name)

	isFresh := false
	repo, err := git.PlainOpen(p)
	if err != nil {
		if err != git.ErrRepositoryNotExists {
			return fmt.Errorf("failed to open repo: %w", err)
		}
		// If we get here, it means the repo just hasn't been cloned yet
		if err := r.clone(); err != nil {
			return fmt.Errorf("failed to clone repo %s: %w", r.Name, err)
		}
		isFresh = true
		repo, err = git.PlainOpen(p)
		if err != nil {
			return fmt.Errorf("failed to open repo after cloning: %w", err)
		}
	}
	// Pull repo, check out branch with upgrade
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %w", err)
	}

	var mainBranch *plumbing.ReferenceName

	// Get all branches
	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to get references: %v", err)
	}
	refs.ForEach(func(ref *plumbing.Reference) error {
		// If main branch is already found, return an error
		if mainBranch != nil {
			return fmt.Errorf("main branch already found")
		}
		// Get name of reference, see if it contains "main" or "master"
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

	// If main branch is not found, return an error
	if mainBranch == nil {
		return ErrNoMainBranch
	}
	mb := *mainBranch

	// Get name of currently checked out branch
	h, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get head: %w", err)
	}

	// If the currently checked out brain is not main,
	// check out the main branch
	if h.Name() != mb {
		if err := w.Checkout(&git.CheckoutOptions{
			Branch: mb,
		}); err != nil {
			return fmt.Errorf("failed to chec out main branch: %w", ErrNoMainBranch)
		}
	}

	// If repo is not freshly cloned, reset any changes (if clean is specified)
	// and pull main branch
	if !isFresh {
		if err := resetAndPull(clean, w); err != nil {
			return fmt.Errorf("failed to reset and pull repo %s: %w", r.Name, err)
		}
	}

	// If branch name to work in has been specified, check it out
	// Create it if it does not exist. By this time,
	// repo should be up to date from main.
	if branchName != "" {
		bn := fmt.Sprintf("refs/heads/%s", branchName)
		if err := w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(bn),
			Create: true,
		}); err != nil {
			return fmt.Errorf("failed to check out branch %s: %w", bn, err)
		}
	}
	return nil
}

func resetAndPull(clean bool, w *git.Worktree) error {
	// If instructed to do a clean sync, revert any local changes
	if clean {
		commit := plumbing.NewHash("HEAD")
		if err := w.Reset(&git.ResetOptions{
			Mode:   git.HardReset,
			Commit: commit,
		}); err != nil {
			return fmt.Errorf("failed to reset repo: %w", err)
		}
	}

	// Pull main branch
	if err := w.Pull(&git.PullOptions{}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to pull repo: %w", err)
		}
	}
	return nil
}

func (r *Repo) clone() error {
	d := r.LocalRepoPath()
	_, err := git.PlainClone(d, false, &git.CloneOptions{
		URL:      r.CloneURL,
		Progress: os.Stdout,
	})

	if err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}
	return nil
}

func (r *Repo) LocalRepoPath() string {
	return filepath.Join(r.Dir, r.Name)
}
