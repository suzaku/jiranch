package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/andygrunwald/go-jira"
	"gopkg.in/yaml.v2"
)

type config struct {
	JiraBaseURL string `yaml:"jiraBaseURL"`
	Username    string
	Token       string
	ShortName   string `yaml:"shortName"`
}

func main() {
	const usage = "Usage: jiranch [config|gen issueID]"
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "config":
		recordConfig()
	case "gen":
		if len(os.Args) != 3 {
			fmt.Println("Please provide an issue ID, for example `jiranch gen XXX-512`")
			os.Exit(1)
		}
		cfg, err := readConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
		}
		issueID := os.Args[2]
		genBranchName(cfg, issueID)
	default:
		fmt.Println(usage)
		os.Exit(1)
	}

}

func readConfig() (config, error) {
	dir, err := getOrCreateProjectDir()
	if err != nil {
		return config{}, err
	}

	configContent, err := ioutil.ReadFile(filepath.Join(dir, "config.yml"))
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Please run `jiranch config` first to generate the config file.")
			os.Exit(1)
		}
		return config{}, err
	}
	var cfg config
	err = yaml.Unmarshal(configContent, &cfg)
	if err != nil {
		return config{}, err
	}
	return cfg, nil
}

func recordConfig() {
	var cfg config
	var err error
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("JIRA base URL: ")
	cfg.JiraBaseURL, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	cfg.JiraBaseURL = strings.TrimSpace(cfg.JiraBaseURL)
	fmt.Print("JIRA user name: ")
	cfg.Username, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	cfg.Username = strings.TrimSpace(cfg.Username)
	fmt.Print("JIRA API token: ")
	cfg.Token, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	fmt.Print("Short name: ")
	cfg.ShortName, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	cfg.ShortName = strings.TrimSpace(cfg.ShortName)

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(err)
	}

	dir, err := getOrCreateProjectDir()
	if err != nil {
		// TODO: handle error
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "config.yml"), data, 0644)
	if err != nil {
		panic(err)
	}
}

func genBranchName(cfg config, issueID string) {
	client, err := createJiraClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create a jira client: %v", err)
		os.Exit(1)
	}
	issue, _, err := client.Issue.Get(issueID, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get issue: %v", err)
		os.Exit(1)
	}
	parts := strings.SplitN(issue.Fields.Summary, " ", 6)
	if len(parts) > 5 {
		parts = parts[:5]
	}

	re := regexp.MustCompile("\\W")
	for i := 0; i < len(parts); i++ {
		parts[i] = string(re.ReplaceAll([]byte(parts[i]), []byte{'_'}))
	}
	suffix := strings.Join(parts, "-")
	fmt.Println(cfg.ShortName + "-" + issueID + "-" + suffix)
}

func getOrCreateProjectDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(usr.HomeDir, ".local/share/jiranch")
	err = os.MkdirAll(dir, 0744)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func createJiraClient(cfg config) (*jira.Client, error) {
	tp := jira.BasicAuthTransport{
		Username: cfg.Username,
		Password: cfg.Token,
	}

	return jira.NewClient(tp.Client(), cfg.JiraBaseURL)
}
