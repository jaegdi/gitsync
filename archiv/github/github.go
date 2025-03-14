package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"gitsync/utils"
)

type GitHubRepo struct {
	Name string `json:"name"`
}

// GitHubResponse represents the response from the GitHub API
type GitHubResponse []GitHubRepo

// ProcessProject processes a GitHub project
func ProcessProject(project, baseDir string, excludeList []string, username, password string) error {
	// Extract the base URL of the GitHub instance
	u, err := url.Parse(project)
	if err != nil {
		return fmt.Errorf("Invalid GitHub project URL: %s", project)
	}
	apiBaseURL := fmt.Sprintf("https://api.%s/repos", u.Host)

	// Extract the user and project from the URL
	parts := strings.Split(u.Path, "/")
	if len(parts) < 3 {
		return fmt.Errorf("Invalid GitHub project URL: %s", project)
	}
	user := parts[1]

	url := fmt.Sprintf("%s/%s", apiBaseURL, user)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %v", err)
	}
	req.SetBasicAuth(username, password)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error fetching GitHub project information: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error fetching GitHub project information: %s", resp.Status)
	}
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	//
	var githubResponse GitHubResponse
	err = json.Unmarshal(body, &githubResponse)
	if err != nil {
		return fmt.Errorf("Error parsing response: %v", err)
	}
	// Clone or pull each repository
	for _, repo := range githubResponse {
		// Skip repositories in the exclude list
		if utils.Contains(excludeList, repo.Name) {
			continue
		}
		repoURL := fmt.Sprintf("https://github.com/%s/%s.git", user, repo.Name)
		// Extract the project name from the URL
		projectName := utils.ExtractProjectFromURL(repoURL)
		// Clone or pull the repository
		err := utils.CloneOrPullRepo(repoURL, filepath.Join(baseDir, projectName), "", "", username, password)
		if err != nil {
			return fmt.Errorf("Error cloning or pulling repository: %v", err)
		}
	}

	return nil
}
