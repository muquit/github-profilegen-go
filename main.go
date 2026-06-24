package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	htmltemplate "html/template" // Import html/template
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template" // Import text/template
	"time"
)

var version = "dev" // overridden at build time via -ldflags

// errRateLimited is returned when the GitHub API reports the rate limit has
// been exhausted, so callers can abort instead of silently producing
// incomplete/incorrect data.
var errRateLimited = fmt.Errorf("GitHub API rate limit exceeded")

// Octicon SVGs (for embedding)
const (
	RepoIconSVG = `<svg class="octicon octicon-repo" viewBox="0 0 16 16" version="1.1" width="16" height="16" aria-hidden="true" style="vertical-align: middle; margin-right: 5px;"><path fill-rule="evenodd" d="M2 2.5A2.5 2.5 0 014.5 0h8.75a.75.75 0 01.75.75v12.5a.75.75 0 01-.75.75h-2.5a.75.75 0 110-1.5h1.75v-2h-8a1 1 0 00-.714 1.7.75.75 0 01-1.072 1.05A2.495 2.495 0 012 11.5v-9zm10.5-1V9h-8c-.356 0-.694.074-1 .208V2.5a1 1 0 011-1h8zM5 12.25v3.25a.25.25 0 00.4.2l1.45-1.087a.25.25 0 01.3 0L8.6 15.7a.25.25 0 00.4-.2v-3.25a.25.25 0 00-.25-.25h-3.5a.25.25 0 00-.25.25z"></path></svg>`
)

// Repository represents a GitHub repository
type Repository struct {
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	Fork        bool      `json:"fork"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PushedAt    time.Time `json:"pushed_at"`
	Homepage    string    `json:"homepage"`
	ForksCount  int       `json:"forks_count"`
	Stargazers  int       `json:"stargazers_count"`
	Source      *struct {
		HTMLURL string `json:"html_url"`
	} `json:"source"`
	HasReleases    bool
	TotalDownloads int
}

// releaseAsset represents a single asset attached to a GitHub release
type releaseAsset struct {
	DownloadCount int `json:"download_count"`
}

// release represents a GitHub release, used to total up asset download counts
type release struct {
	Assets []releaseAsset `json:"assets"`
}

// AICredit holds information about AI assistance
type AICredit struct {
	ImagePath string
	AltText   string
	TitleText string
	Width     string
	Height    string
}

// Config holds the program configuration
type Config struct {
	Username     string
	Token        string // <-- NEW: GitHub Token
	ExcludeFile  string
	PriorityFile string
	AICreditFile string
	ContactFile  string
	OutputFile   string
}

// loadTextFile loads a text file line by line into a slice
func loadTextFile(filename string) ([]string, error) {
	if filename == "" {
		return []string{}, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// loadAICredits loads AI credit information from a file
func loadAICredits(filename string) (map[string]AICredit, error) {
	credits := make(map[string]AICredit)
	if filename == "" {
		return credits, nil
	}
	lines, err := loadTextFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load AI credits from %s: %w", filename, err)
	}
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) >= 6 {
			repoName := strings.TrimSpace(parts[0])
			credits[repoName] = AICredit{
				ImagePath: strings.TrimSpace(parts[1]),
				AltText:   strings.TrimSpace(parts[2]),
				TitleText: strings.TrimSpace(parts[3]),
				Width:     strings.TrimSpace(parts[4]),
				Height:    strings.TrimSpace(parts[5]),
			}
		}
	}
	return credits, nil
}

// formatWithCommas formats an integer with thousands separators, e.g. 1234567 -> "1,234,567"
func formatWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// createRequest creates an authenticated HTTP request
func createRequest(method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "github-profilegen-go/" +version)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token) // <-- Use Token
	}
	return req, nil
}

// fetchRepositories fetches all public repositories using a token
func fetchRepositories(username, token string) ([]Repository, error) {
	var allRepos []Repository
	page := 1
	perPage := 100
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?page=%d&per_page=%d&sort=pushed", username, page, perPage)
		fmt.Printf("Fetching: %s\n", url)
		req, err := createRequest("GET", url, token, nil) // <-- Use createRequest
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, body)
		}

		var repos []Repository
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}
		allRepos = append(allRepos, repos...)
		if len(repos) < perPage {
			break
		}
		page++
		time.Sleep(100 * time.Millisecond)
	}
	return allRepos, nil
}

// fetchLatestRelease checks if a repository has any releases and, if so,
// totals the download counts of all assets across all releases (matching
// what shields.io's github/downloads/.../total badge reports), using a token
func fetchLatestRelease(username, repoName, token string) (bool, int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	total := 0
	hasReleases := false
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?page=%d&per_page=%d", username, repoName, page, perPage)
		req, err := createRequest("GET", url, token, nil)
		if err != nil {
			return false, 0, fmt.Errorf("failed to create request for %s: %w", repoName, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return false, 0, fmt.Errorf("request failed for %s: %w", repoName, err)
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return false, 0, nil
		}
		if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) && resp.Header.Get("X-RateLimit-Remaining") == "0" {
			resp.Body.Close()
			return false, 0, fmt.Errorf("%w (repo: %s)", errRateLimited, repoName)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return false, 0, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, repoName)
		}

		var releases []release
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			resp.Body.Close()
			return false, 0, fmt.Errorf("failed to decode releases response for %s: %w", repoName, err)
		}
		resp.Body.Close()

		if len(releases) == 0 {
			break
		}
		hasReleases = true
		for _, rel := range releases {
			for _, asset := range rel.Assets {
				total += asset.DownloadCount
			}
		}
		if len(releases) < perPage {
			break
		}
		page++
	}

	return hasReleases, total, nil
}

// shouldExcludeRepo checks if a repository should be excluded
func shouldExcludeRepo(repoName string, excludeList []string) bool {
	for _, name := range excludeList {
		if strings.EqualFold(repoName, name) {
			return true
		}
	}
	return false
}

// getPriorityIndex finds the priority index
func getPriorityIndex(repoName string, priorityList []string) int {
	for i, name := range priorityList {
		if strings.EqualFold(repoName, name) {
			return i
		}
	}
	return -1
}

// generateReadme generates the README file
func generateReadme(config Config, repos []Repository, contactInfo []string, aiCredits map[string]AICredit) error {
	const templateText = `
## 📊 

## 📦 Repositories

Here are some of the projects I've worked on. Note: all the badge counts are statically generated because live shields.io badges were rate-limited, and require periodic regeneration.

**Generated on:** {{.Timestamp}}

**Generated with:** [github-profilegen-go]({{.RepoURL}}) {{.Version}}

{{range $index, $repo := .Repos}}
{{if $index}}
<hr>
{{end}}
<h3>{{- $.RepoIconSVG | rawHTML -}}<a href="{{.Repository.HTMLURL}}" target="_blank" rel="noopener noreferrer">{{.Repository.Name}}</a>{{- if .AICredit -}} <a href="#"><img src="{{.AICredit.ImagePath}}" alt="{{.AICredit.AltText}}" title="{{.AICredit.TitleText}}" width="{{.AICredit.Width}}" height="{{.AICredit.Height}}" style="vertical-align: middle; margin-left: 5px;"></a>{{- end -}}</h3>

<p>{{if .Repository.Description}}{{.Repository.Description}}{{else}}<i>No description provided.</i>{{end}}</p>

<p style="font-size: 0.9em;">
{{- if .Repository.Language -}}
<img src="https://img.shields.io/badge/{{.Repository.Language}}-grey?style=flat-square&logo={{.Repository.Language | lower}}&logoColor=white" alt="Language: {{.Repository.Language}}" style="vertical-align: middle;"> 
{{- else -}}
<img src="https://img.shields.io/badge/Language-N/A-grey?style=flat-square" alt="Language: N/A" style="vertical-align: middle;">
{{- end -}}
<img src="https://img.shields.io/badge/Stars-{{commas .Repository.Stargazers}}-blue?style=flat-square" alt="Stars" style="vertical-align: middle;">
<img src="https://img.shields.io/badge/Forks-{{commas .Repository.ForksCount}}-blue?style=flat-square" alt="Forks" style="vertical-align: middle;">
{{- if .Repository.HasReleases -}}
<a href="{{.Repository.HTMLURL}}/releases/latest" target="_blank" rel="noopener noreferrer"><img src="https://img.shields.io/badge/Downloads-{{commas .Repository.TotalDownloads}}-green?style=flat-square" alt="Latest Release Downloads" style="vertical-align: middle;"></a>
{{- end -}}
{{- if .Repository.Fork -}}
<span style="margin-left: 8px; font-style: italic;">(🍴 Forked)</span>
{{- end}}
  <br>
  <small><b>Created</b>: {{.Repository.CreatedAt.Format "Jan 02, 2006"}} | <b>Updated</b>: {{.Repository.UpdatedAt.Format "Jan 02, 2006"}} | <b>Pushed</b>: {{.Repository.PushedAt.Format "Jan 02, 2006"}}</small>
</p>

{{end}}

{{if .ContactInfo}}
## 📫 How to Reach Me

{{range .ContactInfo}}
- {{.}}
{{end}}
{{end}}

---
<p align="right"><small><i>Generated on {{.Timestamp}}</p>
`

	type TemplateRepo struct {
		Repository Repository
		AICredit   *AICredit
	}

	type TemplateData struct {
		Username    string
		Repos       []TemplateRepo
		ContactInfo []string
		Timestamp   string
		RepoIconSVG string
		RepoURL     string
		Version     string
	}

	var templateRepos []TemplateRepo
	for _, repo := range repos {
		var aiCredit *AICredit
		if credit, ok := aiCredits[repo.Name]; ok {
			aiCredit = &credit
		}
		templateRepos = append(templateRepos, TemplateRepo{
			Repository: repo,
			AICredit:   aiCredit,
		})
	}

	data := TemplateData{
		Username:    config.Username,
		Repos:       templateRepos,
		ContactInfo: contactInfo,
		Timestamp:   time.Now().Format(time.RFC1123),
		RepoIconSVG: RepoIconSVG,
		RepoURL:     "https://github.com/muquit/github-profilegen-go",
		Version:     version,
	}

	funcMap := template.FuncMap{
		"lower":  strings.ToLower,
		"commas": formatWithCommas,
		"rawHTML": func(s string) htmltemplate.HTML {
			return htmltemplate.HTML(s)
		},
	}

	tmpl, err := template.New("readme").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	username := flag.String("user", "", "GitHub username (required)")
	token := flag.String("token", "", "GitHub Personal Access Token (or use GITHUB_TOKEN env var)") // <-- NEW: Token Flag
	excludeFile := flag.String("exclude", "", "Path to exclusion list file")
	priorityFile := flag.String("priority", "", "Path to priority list file")
	contactFile := flag.String("contact", "", "Path to contact info file")
	aiCreditFile := flag.String("ai-credits", "", "Path to AI credits file")
	outputFile := flag.String("output", "README.md", "Path to output file")
	flag.Parse()

	if showVersion {
		fmt.Printf("github-profilegen-go %s\n", version)
		os.Exit(0)
	}

	if *username == "" {
		fmt.Println("Error: GitHub username is required. Use the -user flag.")
		flag.Usage()
		os.Exit(1)
	}

	//  ▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼
	//                NEW: Get Token from Flag or Environment
	//  ▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼▼
	githubToken := *token
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		fmt.Println("Warning: No GitHub token provided. Using unauthenticated requests, which have a lower rate limit. Use -token flag or GITHUB_TOKEN env var.")
	}
	//  ▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲

	config := Config{
		Username:     *username,
		Token:        githubToken, // <-- Store Token
		ExcludeFile:  *excludeFile,
		PriorityFile: *priorityFile,
		ContactFile:  *contactFile,
		AICreditFile: *aiCreditFile,
		OutputFile:   *outputFile,
	}

	fmt.Println("Loading configuration...")
	excludeList, err := loadTextFile(config.ExcludeFile)
	if err != nil {
		fmt.Printf("Error loading exclusion file: %v\n", err)
		os.Exit(1)
	}
	priorityList, err := loadTextFile(config.PriorityFile)
	if err != nil {
		fmt.Printf("Error loading priority file: %v\n", err)
		os.Exit(1)
	}
	aiCredits, err := loadAICredits(config.AICreditFile)
	if err != nil {
		fmt.Printf("Error loading AI credits file: %v\n", err)
		os.Exit(1)
	}
	contactInfo, err := loadTextFile(config.ContactFile)
	if err != nil {
		fmt.Printf("Error loading contact file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fetching repositories for %s...\n", config.Username)
	repos, err := fetchRepositories(config.Username, config.Token) // <-- Pass Token
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Fetched %d repositories.\n", len(repos))

	var filteredRepos []Repository
	for _, repo := range repos {
		if !shouldExcludeRepo(repo.Name, excludeList) {
			filteredRepos = append(filteredRepos, repo)
		}
	}
	fmt.Printf("Filtered down to %d repositories.\n", len(filteredRepos))

	fmt.Printf("Checking for releases for %d repos (this may take a while and use API calls)...\n", len(filteredRepos))
	for i := range filteredRepos {
		repo := &filteredRepos[i]
		fmt.Printf("  Checking %s... ", repo.Name)
		has, downloads, err := fetchLatestRelease(config.Username, repo.Name, config.Token)
		if err != nil {
			if errors.Is(err, errRateLimited) {
				fmt.Printf("\nError: %v\n", err)
				fmt.Println("Aborting: refusing to generate a README with incomplete release/download data. Re-run with -token or GITHUB_TOKEN set to a valid GitHub token.")
				os.Exit(1)
			}
			fmt.Printf("Warning: Could not check releases for %s: %v\n", repo.Name, err)
			repo.HasReleases = false
		} else {
			repo.HasReleases = has
			repo.TotalDownloads = downloads
			if has {
				fmt.Println("Found releases.")
			} else {
				fmt.Println("No releases.")
			}
		}
		// You might be able to reduce this sleep or remove it when authenticated,
		// but it's still good practice to be nice to the API.
		time.Sleep(50 * time.Millisecond) // Reduced sleep time
	}
	fmt.Println("Release check complete.")

	sort.Slice(filteredRepos, func(i, j int) bool {
		iPriority := getPriorityIndex(filteredRepos[i].Name, priorityList)
		jPriority := getPriorityIndex(filteredRepos[j].Name, priorityList)

		if iPriority != -1 && jPriority != -1 {
			return iPriority < jPriority
		}
		if iPriority != -1 {
			return true
		}
		if jPriority != -1 {
			return false
		}
		return filteredRepos[i].PushedAt.After(filteredRepos[j].PushedAt)
	})
	fmt.Println("Repositories sorted.")

	fmt.Printf("Generating README.md to %s...\n", config.OutputFile)
	if err := generateReadme(config, filteredRepos, contactInfo, aiCredits); err != nil {
		fmt.Printf("Error generating README: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ README.md generated successfully!")
}
