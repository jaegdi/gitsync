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
	// Flags für den Dateinamen, das Basisverzeichnis und die Authentifizierung definieren
	fileName = flag.String("file", "repos.txt", "Der Name der Repository-Datei")
	baseDir = flag.String("base", ".", "Das Basisverzeichnis für alle Klone")
	username = flag.String("username", "", "Der Benutzername für die Bitbucket-API")
	password = flag.String("password", "", "Das Passwort für die Bitbucket-API")
	passwordFile = flag.String("passwordfile", "", "Der Pfad zu einer Datei, die das Passwort enthält")
	flag.Parse()

	// Passwort aus Datei lesen, falls angegeben
	if *password == "" && *passwordFile != "" {
		data, err := os.ReadFile(*passwordFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fehler beim Lesen der Passwortdatei:", err)
			os.Exit(1)
		}
		*password = strings.TrimSpace(string(data))
	}

	// Basisverzeichnis erstellen, falls es nicht existiert
	if _, err := os.Stat(*baseDir); os.IsNotExist(err) {
		err := os.MkdirAll(*baseDir, 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fehler beim Erstellen des Basisverzeichnisses:", err)
			os.Exit(1)
		}
	}

	// Log-Datei im Basisverzeichnis öffnen
	logFilePath := filepath.Join(*baseDir, "gitsync.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Öffnen der Log-Datei:", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
}

func main() {
	// Datei öffnen
	file, err := os.Open(*fileName)
	if err != nil {
		log.Println("Fehler beim Öffnen der Datei:", err)
		fmt.Fprintln(os.Stderr, "Fehler beim Öffnen der Datei:", err)
		return
	}
	defer file.Close()

	// Scanner zum Lesen der Datei
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			log.Println("Ungültiges Format in der Datei:", line)
			fmt.Fprintln(os.Stderr, "Ungültiges Format in der Datei:", line)
			continue
		}
		typ, name := parts[0], parts[1]
		excludeList := []string{}
		if len(parts) >= 3 && parts[2] == "i" {
			excludeList = strings.Split(parts[3], ",")
			fmt.Println("Ausschlussliste:", excludeList)
		}
		if typ == "p" {
			// Bitbucket-Projekt verarbeiten
			log.Println("Verarbeite Bitbucket-Projekt:", name)
			fmt.Println("Verarbeite Bitbucket-Projekt:", name)
			err := processBitbucketProject(name, *baseDir, excludeList)
			if err != nil {
				log.Println("Fehler beim Verarbeiten des Bitbucket-Projekts:", err)
				fmt.Fprintln(os.Stderr, "Fehler beim Verarbeiten des Bitbucket-Projekts:", err)
			}
		} else if typ == "r" {
			// Repository klonen oder pullen
			log.Println("Klonen oder Pullen des Repositories:", name)
			fmt.Println("Klonen oder Pullen des Repositories:", name)
			project := extractProjectFromURL(name)
			err := cloneOrPullRepo(name, filepath.Join(*baseDir, project))
			if err != nil {
				log.Println("Fehler beim Klonen oder Pullen des Repositories:", err)
				fmt.Fprintln(os.Stderr, "Fehler beim Klonen oder Pullen des Repositories:", err)
			}
		} else {
			log.Println("Unbekannter Typ in der Datei:", typ)
			fmt.Fprintln(os.Stderr, "Unbekannter Typ in der Datei:", typ)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Fehler beim Lesen der Datei:", err)
		fmt.Fprintln(os.Stderr, "Fehler beim Lesen der Datei:", err)
	}
}

func processBitbucketProject(project, baseDir string, excludeList []string) error {
	// Extrahiere die Basis-URL der Bitbucket-Instanz
	u, err := url.Parse(project)
	if err != nil {
		return fmt.Errorf("Ungültige Bitbucket-Projekt-URL: %s", project)
	}
	apiBaseURL := fmt.Sprintf("https://%s/rest/api/1.0/projects", u.Host)

	// Extrahiere den Workspace und das Projekt aus der URL
	parts := strings.Split(u.Path, "/")
	if len(parts) < 3 {
		return fmt.Errorf("Ungültige Bitbucket-Projekt-URL: %s", project)
	}
	workspace := parts[2]

	url := fmt.Sprintf("%s/%s/repos", apiBaseURL, workspace)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Anfrage: %v", err)
	}
	req.SetBasicAuth(*username, *password)

	// Debugging-Informationen hinzufügen
	log.Println("Benutzername:", *username)
	log.Println("Passwort:", strings.Repeat("*", len(*password)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen der Bitbucket-Projektinformationen: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Println("Antwort:", string(body))
		return fmt.Errorf("Fehler beim Abrufen der Bitbucket-Projektinformationen: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Fehler beim Lesen der Antwort: %v", err)
	}

	var bitbucketResponse BitbucketResponse
	err = json.Unmarshal(body, &bitbucketResponse)
	if err != nil {
		return fmt.Errorf("Fehler beim Parsen der Antwort: %v", err)
	}

	for _, repo := range bitbucketResponse.Values {
		if contains(excludeList, repo.Slug) {
			log.Println("Überspringe Repository:", repo.Slug)
			fmt.Println("Überspringe Repository:", repo.Slug)
			continue
		}
		repoURL := fmt.Sprintf("ssh://git@%s:7999/%s/%s.git", u.Host, workspace, repo.Slug)
		log.Println("Klonen oder Pullen des Repositories:", repoURL)
		fmt.Println("Klonen oder Pullen des Repositories:", repoURL)
		err := cloneOrPullRepo(repoURL, filepath.Join(baseDir, workspace))
		if err != nil {
			log.Println("Fehler beim Klonen oder Pullen des Repositories:", err)
			fmt.Fprintln(os.Stderr, "Fehler beim Klonen oder Pullen des Repositories:", err)
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
		auth = &gitHttp.BasicAuth{Username: *username, Password: *password} // Falls Authentifizierung erforderlich ist
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
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		// Repository klonen
		log.Println("Klonen des Repositories:", url)
		fmt.Println("Klonen des Repositories:", url)
		_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
			Auth:     auth,
		})
		return err
	} else {
		// Repository pullen
		log.Println("Pullen des Repositories:", url)
		fmt.Println("Pullen des Repositories:", url)
		repo, err := git.PlainOpen(cloneDir)
		if err != nil {
			return fmt.Errorf("Fehler beim Öffnen des Repositories: %v", err)
		}
		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("Fehler beim Abrufen des Worktrees: %v", err)
		}
		err = w.Pull(&git.PullOptions{
			RemoteName: "origin",
			Auth:       auth,
			Progress:   os.Stdout,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("Fehler beim Pullen des Repositories: %v", err)
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
