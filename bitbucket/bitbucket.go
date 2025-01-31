package bitbucket

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

type BitbucketRepo struct {
	Slug string `json:"slug"`
}

type BitbucketResponse struct {
	Values []BitbucketRepo `json:"values"`
}

func ProcessProject(project, baseDir string, excludeList []string, username, password string) error {
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
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error fetching Bitbucket project information: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		if utils.Contains(excludeList, repo.Slug) {
			continue
		}
		repoURL := fmt.Sprintf("ssh://git@%s:7999/%s/%s.git", u.Host, workspace, repo.Slug)
		err := utils.CloneOrPullRepo(repoURL, filepath.Join(baseDir, workspace), "", "", username, password)
		if err != nil {
			return fmt.Errorf("Error cloning or pulling repository: %v", err)
		}
	}

	return nil
}
