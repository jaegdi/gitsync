package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gitsync/utils"
	"gitsync/vcs"

	"gopkg.in/yaml.v2"
)

type Repo struct {
	VCS      string   `yaml:"vcs"`                // Version Control System
	Type     string   `yaml:"type"`               // Type of repo (project or repo)
	URL      string   `yaml:"url"`                // URL of the repo
	Ignore   []string `yaml:"ignore,omitempty"`   // optional List of directories to ignore
	User     string   `yaml:"user,omitempty"`     // optionale Username for the API, if differs from default username
	Password string   `yaml:"password,omitempty"` // Password for the API if differs from default password, can be a command to get the password or 'ask' to get it interactive
	Branch   string   `yaml:"branch,omitempty"`   // optional Branch to checkout
	Tag      string   `yaml:"tag,omitempty"`      // optional Tag to checkout
}

type Config struct {
	Repos []Repo `yaml:"repos"`
}

var (
	repoListName *string
	baseDir      *string
	username     *string
	password     *string
	passwordFile *string
)

func init() {
	// Define flags for the file name, base directory, and authentication
	repoListName = flag.String("repolist", "repos.yaml", "The name of the yaml file with the repo list")
	baseDir = flag.String("basedir", ".", "The base directory for all clones")
	username = flag.String("username", "", "The username for the API")
	password = flag.String("password", "", "The default password for the API of the git server, for different password per repo insert them in the yaml repo-list")
	passwordFile = flag.String("passwordfile", "", "The path to a file containing the default password for the api of the git server")
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
	file, err := os.Open(*repoListName)
	if err != nil {
		log.Println("Error opening file:", err)
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		return
	}
	defer file.Close()

	// Read YAML config file
	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Println("Error decoding YAML file:", err)
		fmt.Fprintln(os.Stderr, "Error decoding YAML file:", err)
		return
	}

	// Process each repository
	for _, repo := range config.Repos {
		excludeList := repo.Ignore
		if len(excludeList) > 0 {
			fmt.Println("Exclude list:", excludeList)
		}
		if repo.Type == "project" {
			// Process project
			log.Println("\n-----------------------------------\nProcessing project:", repo.URL)
			fmt.Println("\n-----------------------------------\nProcessing project:", repo.URL)
			err := vcs.ProcessProject(repo.VCS, repo.URL, *baseDir, excludeList, *username, *password, repo.Branch, repo.Tag)
			if err != nil {
				log.Println("Error processing project:", err)
				fmt.Fprintln(os.Stderr, "Error processing project:", err)
			}
			fmt.Println("\n-----------------------------------")
		} else if repo.Type == "repo" {
			// Clone or pull repository
			log.Println("\nCloning or pulling repository:", repo.URL)
			fmt.Println("\nCloning or pulling repository:", repo.URL)
			project := utils.ExtractProjectFromURL(repo.URL)
			err := utils.CloneOrPullRepo(repo.URL, filepath.Join(*baseDir, project), repo.User, repo.Password, *username, *password, repo.Branch, repo.Tag)
			if err != nil {
				log.Println("Error cloning or pulling repository:", err)
				fmt.Fprintln(os.Stderr, "Error cloning or pulling repository:", err)
			}
		} else {
			log.Println("Unknown type in file:", repo.Type)
			fmt.Fprintln(os.Stderr, "Unknown type in file:", repo.Type)
		}
	}
}
