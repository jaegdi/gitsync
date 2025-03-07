package utils

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/term"
)

// ExtractProjectFromURL extracts the project name from a URL
func ExtractProjectFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	parts := strings.Split(u.Host, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	subdomain := parts[0]
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) >= 3 {
		return fmt.Sprintf("%s-%s", subdomain, pathParts[len(pathParts)-2])
	}
	return fmt.Sprintf("%s-%s", subdomain, pathParts[0])
}

// CloneOrPullRepo clones or pulls a repository
func CloneOrPullRepo(url, dir, repoUser, repoPassword, username, password, branch, tag string) error {
	auth, err := getAuthMethod(url, repoUser, repoPassword, username, password)
	if err != nil {
		return err
	}

	cloneDir := filepath.Join(dir, filepath.Base(url))
	logFile, err := os.OpenFile("gitsync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	var repo *git.Repository
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		// Clone repository
		fmt.Println("\nCloning repository:", url, "to", cloneDir)
		logFile.WriteString(fmt.Sprintf("\nCloning repository: %s\n", url))
		cloneOptions := &git.CloneOptions{
			URL:      url,
			Progress: logFile,
			Auth:     auth,
		}
		if branch != "" {
			cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(branch)
			cloneOptions.SingleBranch = true
		}
		repo, err = git.PlainClone(cloneDir, false, cloneOptions)
		if err != nil {
			return fmt.Errorf("Error cloning repository: %v", err)
		}
	} else {
		// Pull repository
		fmt.Println("\nPulling repository:", url)
		logFile.WriteString(fmt.Sprintf("\nPulling repository: %s in %s\n", url, cloneDir))
		repo, err = git.PlainOpen(cloneDir)
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
			Progress:   logFile,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("Error pulling repository: %v", err)
		}
	}

	// Checkout tag if specified
	if tag != "" {
		err = CheckoutBranchOrTag(repo, "", tag)
		if err != nil {
			return fmt.Errorf("Error checking out tag: %v", err)
		}
	}

	return nil
}

// getAuthMethod returns the appropriate authentication method based on the URL and credentials
func getAuthMethod(url, repoUser, repoPassword, username, password string) (transport.AuthMethod, error) {
	// Execute command if password contains spaces
	if strings.Contains(repoPassword, " ") {
		cmd := exec.Command("sh", "-c", repoPassword)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("Error executing password command: %v", err)
		}
		repoPassword = strings.TrimSpace(out.String())
	}

	// Ask for password if the value is 'ask'
	if repoPassword == "ask" {
		fmt.Printf("Enter password for %s user %s: ", url, username)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("Error reading password: %v", err)
		}
		repoPassword = string(bytePassword)
	}

	if repoUser != "" && repoPassword != "" {
		return &gitHttp.BasicAuth{Username: repoUser, Password: repoPassword}, nil
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return &gitHttp.BasicAuth{Username: username, Password: password}, nil // If authentication is required
	} else if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("Error determining user directory: %v", err)
		}
		sshKeyPath := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
		sshAuth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
		if err != nil {
			return nil, fmt.Errorf("Error loading SSH key: %v", err)
		}
		return sshAuth, nil
	}
	return nil, fmt.Errorf("Unsupported URL scheme: %s", url)
}

// Contains checks if a string slice contains a substring
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

// CheckoutBranchOrTag checks out the specified branch or tag
func CheckoutBranchOrTag(repo *git.Repository, branch, tag string) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("Error retrieving worktree: %v", err)
	}

	if branch != "" {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
		})
		if err != nil {
			return fmt.Errorf("Error checking out branch: %v", err)
		}
	} else if tag != "" {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewTagReferenceName(tag),
		})
		if err != nil {
			return fmt.Errorf("Error checking out tag: %v", err)
		}
	}

	return nil
}
