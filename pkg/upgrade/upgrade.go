package upgrade

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed scripts/npm-upgrade.sh
var npmUpgrade string

func Upgrade(dir string) error {
	// Only NPM currently supported
	return tryNpm(dir)
}

func tryNpm(dir string) error {
	if !isNpmPkg(dir) {
		return nil
	}

	cmd := exec.Command("/bin/sh")
	cmd.Stdin = strings.NewReader(npmUpgrade)
	cmd.Dir = dir

	o, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to update dependencies: %w (%s)", err, o)
	}
	return nil
}
func isNpmPkg(dir string) bool {
	repoPath := dir
	pkgJson := filepath.Join(repoPath, "package.json")
	fi, err := os.Stat(pkgJson)
	if os.IsNotExist(err) {
		return false
	}
	return !fi.IsDir()
}
