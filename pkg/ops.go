package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const defaultApiURL = "https://api.github.com"

type OrgCloner struct {
	apiURL  string
	orgName string
	syncDir string
	token   string
	force   bool
}

func NewOrgCloner(orgName string, options ...func(cloner *OrgCloner)) *OrgCloner {
	cloner := &OrgCloner{
		apiURL:  defaultApiURL,
		orgName: orgName,
	}
	for _, opt := range options {
		opt(cloner)
	}
	return cloner
}

func WithApiUrl(url string) func(cloner *OrgCloner) {
	return func(s *OrgCloner) {
		s.apiURL = url
	}
}

func WithToken(token string) func(cloner *OrgCloner) {
	return func(s *OrgCloner) {
		s.token = token
	}
}

func WithSyncDir(dir string) func(cloner *OrgCloner) {
	return func(s *OrgCloner) {
		s.syncDir = dir
	}
}

func WithForce(force bool) func(cloner *OrgCloner) {
	return func(s *OrgCloner) {
		s.force = force
	}
}

func (o *OrgCloner) SyncAllRepos() ([]*Repo, error) {
	allRepos, err := o.getAllRepos()
	if err != nil {
		return nil, err
	}
	dir := o.syncDir
	if dir == "" {
		path, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working dir: %w", err)
		}
		dir = path
	}

	for _, r := range allRepos {
		if err := r.Sync(dir, o.force); err != nil {
			return nil, fmt.Errorf("failed to sync repo %s: %v", r.Name, err)
		}
	}
	return allRepos, nil
}

func (o *OrgCloner) getAllRepos() ([]*Repo, error) {
	// Get the list of repositories in the organization
	url := fmt.Sprintf("%s/orgs/%s/repos?per_page=100", o.apiURL, o.orgName)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	if o.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.token))
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedRepoFetch, err)
	}
	sc := res.StatusCode
	if sc != 200 {
		return nil, fmt.Errorf("expected status code 200, got %d: %w", sc, ErrFailedRepoFetch)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub response body: %w", err)
	}

	var allRepos []*Repo
	if err := json.Unmarshal(body, &allRepos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GitHub response body into slice of repos: %w", err)
	}
	return allRepos, nil
}
