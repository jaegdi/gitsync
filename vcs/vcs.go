package vcs

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

type Repo struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Response struct {
	Values []Repo `json:"values"`
}

func ProcessProject(vcsType, project, baseDir string, excludeList []string, username, password, branch, tag string) error {
	u, err := url.Parse(project)
	if err != nil {
		return fmt.Errorf("Invalid project URL: %s", project)
	}

	apiBaseURL, user, workspace, err := getAPIBaseURL(vcsType, u)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", apiBaseURL, nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %v", err)
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error fetching project information: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error fetching project information: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("Error parsing response: %v", err)
	}

	for _, repo := range response.Values {
		repoName := getRepoName(vcsType, repo)
		if utils.Contains(excludeList, repoName) {
			continue
		}
		repoURL, err := getRepoURL(vcsType, u, user, workspace, repoName)
		if err != nil {
			return err
		}
		projectName := utils.ExtractProjectFromURL(repoURL)
		err = utils.CloneOrPullRepo(repoURL, filepath.Join(baseDir, projectName), "", "", username, password, branch, tag)
		if err != nil {
			return fmt.Errorf("Error cloning or pulling repository: %v", err)
		}
	}

	return nil
}

func getAPIBaseURL(vcsType string, u *url.URL) (string, string, string, error) {
	var apiBaseURL, user, workspace string
	switch vcsType {
	case "github":
		apiBaseURL = fmt.Sprintf("https://api.%s/repos", u.Host)
		parts := strings.Split(u.Path, "/")
		if len(parts) < 3 {
			return "", "", "", fmt.Errorf("Invalid GitHub project URL: %s", u.String())
		}
		user = parts[1]
		apiBaseURL = fmt.Sprintf("%s/%s", apiBaseURL, user)
	case "bitbucket":
		apiBaseURL = fmt.Sprintf("https://%s/rest/api/1.0/projects", u.Host)
		parts := strings.Split(u.Path, "/")
		if len(parts) < 3 {
			return "", "", "", fmt.Errorf("Invalid Bitbucket project URL: %s", u.String())
		}
		workspace = parts[2]
		apiBaseURL = fmt.Sprintf("%s/%s/repos", apiBaseURL, workspace)
	default:
		return "", "", "", fmt.Errorf("Unsupported VCS type: %s", vcsType)
	}
	return apiBaseURL, user, workspace, nil
}

func getRepoName(vcsType string, repo Repo) string {
	if vcsType == "bitbucket" {
		return repo.Slug
	}
	return repo.Name
}

func getRepoURL(vcsType string, u *url.URL, user, workspace, repoName string) (string, error) {
	switch vcsType {
	case "github":
		return fmt.Sprintf("https://github.com/%s/%s.git", user, repoName), nil
	case "bitbucket":
		return fmt.Sprintf("ssh://git@%s:7999/%s/%s.git", u.Host, workspace, repoName), nil
	default:
		return "", fmt.Errorf("Unsupported VCS type: %s", vcsType)
	}
}
