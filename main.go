package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gitsync/bitbucket"
	"gitsync/github"
	"gitsync/utils"

	"gopkg.in/yaml.v2"
)

type Repo struct {
	VCS      string   `yaml:"vcs"`                // Version Control System
	Type     string   `yaml:"type"`               // Type of repo (project or repo)
	URL      string   `yaml:"url"`                // URL of the repo
	Ignore   []string `yaml:"ignore,omitempty"`   // List of directories to ignore
	User     string   `yaml:"user,omitempty"`     // Username for the API
	Password string   `yaml:"password,omitempty"` // Password for the API, can be a command to get the password or 'ask' to get it interactive
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
		if repo.VCS == "bitbucket" {
			if repo.Type == "project" {
				// Process Bitbucket project
				log.Println("\n-----------------------------------\nProcessing Bitbucket project:", repo.URL)
				fmt.Println("\n-----------------------------------\nProcessing Bitbucket project:", repo.URL)
				err := bitbucket.ProcessProject(repo.URL, *baseDir, excludeList, *username, *password)
				if err != nil {
					log.Println("Error processing Bitbucket project:", err)
					fmt.Fprintln(os.Stderr, "Error processing Bitbucket project:", err)
				}
			} else if repo.Type == "repo" {
				// Clone or pull Bitbucket repository
				log.Println("\nCloning or pulling Bitbucket repository:", repo.URL)
				fmt.Println("\nCloning or pulling Bitbucket repository:", repo.URL)
				project := utils.ExtractProjectFromURL(repo.URL)
				err := utils.CloneOrPullRepo(repo.URL, filepath.Join(*baseDir, project), repo.User, repo.Password, *username, *password)
				if err != nil {
					log.Println("Error cloning or pulling Bitbucket repository:", err)
					fmt.Fprintln(os.Stderr, "Error cloning or pulling Bitbucket repository:", err)
				}
			} else {
				log.Println("Unknown type in file:", repo.Type)
				fmt.Fprintln(os.Stderr, "Unknown type in file:", repo.Type)
			}
		} else if repo.VCS == "github" {
			if repo.Type == "project" {
				// Process GitHub project
				log.Println("\nProcessing GitHub project:", repo.URL)
				fmt.Println("\nProcessing GitHub project:", repo.URL)
				err := github.ProcessProject(repo.URL, *baseDir, excludeList, *username, *password)
				if err != nil {
					log.Println("Error processing GitHub project:", err)
					fmt.Fprintln(os.Stderr, "Error processing GitHub project:", err)
				}
			} else if repo.Type == "repo" {
				// Clone or pull GitHub repository
				log.Println("\nCloning or pulling GitHub repository:", repo.URL)
				fmt.Println("\nCloning or pulling GitHub repository:", repo.URL)
				project := utils.ExtractProjectFromURL(repo.URL)
				err := utils.CloneOrPullRepo(repo.URL, filepath.Join(*baseDir, project), repo.User, repo.Password, *username, *password)
				if err != nil {
					log.Println("Error cloning or pulling GitHub repository:", err)
					fmt.Fprintln(os.Stderr, "Error cloning or pulling GitHub repository:", err)
				}
			} else {
				// Unknown repo type
				log.Println("Unknown type in file:", repo.Type)
				fmt.Fprintln(os.Stderr, "Unknown type in file:", repo.Type)
			}
		} else {
			log.Println("Unknown VCS in file:", repo.VCS)
			fmt.Fprintln(os.Stderr, "Unknown VCS in file:", repo.VCS)
		}
	}
}
