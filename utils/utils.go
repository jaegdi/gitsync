package utils

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func ExtractProjectFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[len(parts)-2]
	}
	return "unknown"
}

func CloneOrPullRepo(url, dir, repoUser, repoPassword, username, password string) error {
	var auth transport.AuthMethod

	if repoUser != "" && repoPassword != "" {
		auth = &gitHttp.BasicAuth{Username: repoUser, Password: repoPassword}
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		auth = &gitHttp.BasicAuth{Username: username, Password: password} // If authentication is required
	} else if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("Error determining user directory: %v", err)
		}
		sshKeyPath := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
		sshAuth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
		if err != nil {
			return fmt.Errorf("Error loading SSH key: %v", err)
		}
		auth = sshAuth
	}

	cloneDir := filepath.Join(dir, filepath.Base(url))
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		// Clone repository
		fmt.Println("Cloning repository:", url)
		_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
			Auth:     auth,
		})
		return err
	} else {
		// Pull repository
		fmt.Println("Pulling repository:", url)
		repo, err := git.PlainOpen(cloneDir)
		if err != nil {
			return fmt.Errorf("Error opening repository: %v", err)
		}
		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("Error retrieving worktree: %v", err)
		}
		err = w.Pull(&git.PullOptions{
			RemoteName: "origin",
			Auth:       auth,
			Progress:   os.Stdout,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("Error pulling repository: %v", err)
		}
		return nil
	}
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}
