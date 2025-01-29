package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
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

func main() {
	// Flag für den Dateinamen definieren
	fileName := flag.String("file", "repos.txt", "Der Name der Repository-Datei")
	flag.Parse()

	// Datei öffnen
	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Println("Fehler beim Öffnen der Datei:", err)
		return
	}
	defer file.Close()

	// Scanner zum Lesen der Datei
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			fmt.Println("Ungültiges Format in der Datei:", line)
			continue
		}
		typ, name := parts[0], parts[1]
		if typ == "p" {
			// Bitbucket-Projekt verarbeiten
			err := processBitbucketProject(name)
			if err != nil {
				fmt.Println("Fehler beim Verarbeiten des Bitbucket-Projekts:", err)
			}
		} else if typ == "r" {
			// Repository klonen
			err := cloneRepo(name, ".")
			if err != nil {
				fmt.Println("Fehler beim Klonen des Repositories:", err)
			}
		} else {
			fmt.Println("Unbekannter Typ in der Datei:", typ)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Fehler beim Lesen der Datei:", err)
	}
}

func processBitbucketProject(project string) error {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s", project)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen der Bitbucket-Projektinformationen: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Fehler beim Abrufen der Bitbucket-Projektinformationen: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Fehler beim Lesen der Antwort: %v", err)
	}

	var repos []BitbucketRepo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		return fmt.Errorf("Fehler beim Parsen der Antwort: %v", err)
	}

	for _, repo := range repos {
		repoURL := fmt.Sprintf("git@bitbucket.org:%s/%s.git", project, repo.Slug)
		err := cloneRepo(repoURL, project)
		if err != nil {
			fmt.Println("Fehler beim Klonen des Repositories:", err)
		}
	}

	return nil
}

func cloneRepo(url, dir string) error {
	var auth transport.AuthMethod

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		auth = &gitHttp.BasicAuth{Username: "your-username", Password: "your-password"} // Falls Authentifizierung erforderlich ist
	} else if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("Fehler beim Ermitteln des Benutzerverzeichnisses: %v", err)
		}
		sshKeyPath := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
		sshAuth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
		if err != nil {
			return fmt.Errorf("Fehler beim Laden des SSH-Schlüssels: %v", err)
		}
		auth = sshAuth
	}

	cloneDir := filepath.Join(dir, filepath.Base(url))
	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
		Auth:     auth,
	})
	return err
}
