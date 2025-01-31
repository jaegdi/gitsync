package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

type BitbucketRepo struct {
	Slug string `json:"slug"`
}

type BitbucketResponse struct {
	Values []BitbucketRepo `json:"values"`
}

var (
	fileName     *string
	baseDir      *string
	username     *string
	password     *string
	passwordFile *string
)

func init() {
	// Define flags for the file name, base directory, and authentication
	fileName = flag.String("file", "repos.txt", "The name of the repository file")
	baseDir = flag.String("base", ".", "The base directory for all clones")
	username = flag.String("username", "", "The username for the Bitbucket API")
	password = flag.String("password", "", "The password for the Bitbucket API")
	passwordFile = flag.String("passwordfile", "", "The path to a file containing the password")
	flag.Parse()

	// Read password from file if specified
	if *password == "" && *passwordFile != "" {
		data, err := os.ReadFile(*passwordFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading password file:", err)
			os.Exit(1)
		}
		*password = strings.TrimSpace(string(data))
	}

	// Create base directory if it does not exist
	if _, err := os.Stat(*baseDir); os.IsNotExist(err) {
		err := os.MkdirAll(*baseDir, 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating base directory:", err)
			os.Exit(1)
		}
	}

	// Open log file in base directory
	logFilePath := filepath.Join(*baseDir, "gitsync.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening log file:", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
}

func main() {
	// Open file
	file, err := os.Open(*fileName)
	if err != nil {
		log.Println("Error opening file:", err)
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		return
	}
	defer file.Close()

	// Scanner to read the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			log.Println("Invalid format in file:", line)
			fmt.Fprintln(os.Stderr, "Invalid format in file:", line)
			continue
		}
		typ, name := parts[0], parts[1]
		excludeList := []string{}
		if len(parts) >= 3 && parts[2] == "i" {
			excludeList = strings.Split(parts[3], ",")
			fmt.Println("Exclude list:", excludeList)
		}
		if typ == "p" {
			// Process Bitbucket project
			log.Println("Processing Bitbucket project:", name)
			fmt.Println("Processing Bitbucket project:", name)
			err := processBitbucketProject(name, *baseDir, excludeList)
			if err != nil {
				log.Println("Error processing Bitbucket project:", err)
				fmt.Fprintln(os.Stderr, "Error processing Bitbucket project:", err)
			}
		} else if typ == "r" {
			// Clone or pull repository
			log.Println("Cloning or pulling repository:", name)
			fmt.Println("Cloning or pulling repository:", name)
			project := extractProjectFromURL(name)
			err := cloneOrPullRepo(name, filepath.Join(*baseDir, project))
			if err != nil {
				log.Println("Error cloning or pulling repository:", err)
				fmt.Fprintln(os.Stderr, "Error cloning or pulling repository:", err)
			}
		} else {
			log.Println("Unknown type in file:", typ)
			fmt.Fprintln(os.Stderr, "Unknown type in file:", typ)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
		fmt.Fprintln(os.Stderr, "Error reading file:", err)
	}
}

func processBitbucketProject(project, baseDir string, excludeList []string) error {
	// Extract the base URL of the Bitbucket instance
	u, err := url.Parse(project)
	if err != nil {
		return fmt.Errorf("Invalid Bitbucket project URL: %s", project)
	}
	apiBaseURL := fmt.Sprintf("https://%s/rest/api/1.0/projects", u.Host)

	// Extract the workspace and project from the URL
	parts := strings.Split(u.Path, "/")
	if len(parts) < 3 {
		return fmt.Errorf("Invalid Bitbucket project URL: %s", project)
	}
	workspace := parts[2]

	url := fmt.Sprintf("%s/%s/repos", apiBaseURL, workspace)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %v", err)
	}
	req.SetBasicAuth(*username, *password)

	// Add debugging information
	log.Println("Username:", *username)
	log.Println("Password:", strings.Repeat("*", len(*password)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error fetching Bitbucket project information: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Println("Response:", string(body))
		return fmt.Errorf("Error fetching Bitbucket project information: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}

	var bitbucketResponse BitbucketResponse
	err = json.Unmarshal(body, &bitbucketResponse)
	if err != nil {
		return fmt.Errorf("Error parsing response: %v", err)
	}

	for _, repo := range bitbucketResponse.Values {
		if contains(excludeList, repo.Slug) {
			log.Println("Skipping repository:", repo.Slug)
			fmt.Println("Skipping repository:", repo.Slug)
			continue
		}
		repoURL := fmt.Sprintf("ssh://git@%s:7999/%s/%s.git", u.Host, workspace, repo.Slug)
		log.Println("Cloning or pulling repository:", repoURL)
		fmt.Println("Cloning or pulling repository:", repoURL)
		err := cloneOrPullRepo(repoURL, filepath.Join(baseDir, workspace))
		if err != nil {
			log.Println("Error cloning or pulling repository:", err)
			fmt.Fprintln(os.Stderr, "Error cloning or pulling repository:", err)
		}
	}

	return nil
}

func extractProjectFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[len(parts)-2]
	}
	return "unknown"
}

func cloneOrPullRepo(url, dir string) error {
	var auth transport.AuthMethod

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		auth = &gitHttp.BasicAuth{Username: *username, Password: *password} // If authentication is required
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
		log.Println("Cloning repository:", url)
		fmt.Println("Cloning repository:", url)
		_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
			Auth:     auth,
		})
		return err
	} else {
		// Pull repository
		log.Println("Pulling repository:", url)
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}
