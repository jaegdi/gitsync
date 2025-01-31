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

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// ExtractProjectFromURL extracts the project name from a URL
func ExtractProjectFromURL(rawURL string) string {
	fmt.Println("Extracting project from URL:", rawURL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	// Extract subdomain from host
	parts := strings.Split(u.Host, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	subdomain := parts[0]
	fmt.Println("Subdomain:", subdomain)

	// Extract project name from path
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) >= 3 {
		fmt.Println("Project Dir Name:", fmt.Sprintf("%s-%s", subdomain, pathParts[len(pathParts)-2]))
		return fmt.Sprintf("%s-%s", subdomain, pathParts[len(pathParts)-2])
	}
	fmt.Println("Project Dir Name:", fmt.Sprintf("%s-%s", subdomain, pathParts[0]))
	return fmt.Sprintf("%s-%s", subdomain, pathParts[0])
}

// CloneOrPullRepo clones or pulls a repository
func CloneOrPullRepo(url, dir, repoUser, repoPassword, username, password string) error {
	var auth transport.AuthMethod

	// Execute command if password contains spaces
	if strings.Contains(repoPassword, " ") {
		cmd := exec.Command("sh", "-c", repoPassword)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Error executing password command: %v", err)
		}
		repoPassword = strings.TrimSpace(out.String())
	}

	// Ask for password if the value is 'ask'
	if repoPassword == "ask" {
		fmt.Printf("Enter password for %s user %s: ", url, username)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("Error reading password: %v", err)
		}
		repoPassword = string(bytePassword)
	}
	// Set authentication method based on URL
	if repoUser != "" && repoPassword != "" {
		auth = &gitHttp.BasicAuth{Username: repoUser, Password: repoPassword}
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		auth = &gitHttp.BasicAuth{Username: username, Password: password} // If authentication is required
	} else if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		// Load SSH key if URL is SSH
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
	// Clone or pull repository
	cloneDir := filepath.Join(dir, filepath.Base(url))
	logFile, err := os.OpenFile("gitsync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("Error opening log file: %v", err)
	}
	defer logFile.Close()
	// Clone repository if it doesn't exist or pull it if it does
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		// Clone repository
		fmt.Println("\nCloning repository:", url)
		logFile.WriteString(fmt.Sprintf("\nCloning repository: %s\n", url))
		_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:      url,
			Progress: logFile,
			Auth:     auth,
		})
		return err
	} else {
		// Pull repository
		fmt.Println("\nPulling repository:", url)
		logFile.WriteString(fmt.Sprintf("\nPulling repository: %s\n", url))
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
			Progress:   logFile,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("Error pulling repository: %v", err)
		}
		return nil
	}
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
