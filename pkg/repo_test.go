package pkg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSyncRepo(t *testing.T) {
	cases := []struct {
		name    string
		repo    *Repo
		force   bool
		wantErr error
	}{
		{
			name: "success",
			repo: &Repo{
				ID:   0,
				Name: "testrepo",
			},
		},
		{
			name: "nonnpm",
			repo: &Repo{
				ID:   0,
				Name: "nonnpm",
			},
			wantErr: ErrNoMainBranch,
		},
	}

	dir := "testdata"
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotErr := tc.repo.Sync(dir, tc.force)
			assert.ErrorIs(t, gotErr, tc.wantErr)
		})
	}
}
