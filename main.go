package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

const (
	VERSION = "1.0.2"
)

// Repository represents a GitHub repository with the fields we're interested in
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
}

// AICredit holds information about AI assistance for a repository
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
	ExcludeFile  string
	PriorityFile string
	AICreditFile string
	OutputFile   string
}

// loadTextFile loads a text file line by line into a slice
func loadTextFile(filename string) ([]string, error) {
	if filename == "" {
		return []string{}, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
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

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// loadAICredits loads AI credit information from a file
func loadAICredits(filename string) (map[string]AICredit, error) {
	credits := make(map[string]AICredit)

	if filename == "" {
		return credits, nil
	}

	lines, err := loadTextFile(filename)
	if err != nil {
		return nil, err
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

// fetchRepositories fetches all public repositories for a given username
func fetchRepositories(username string) ([]Repository, error) {
	var allRepos []Repository
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?page=%d&per_page=%d", username, page, perPage)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// Add a user agent to avoid GitHub API limitations
		req.Header.Set("User-Agent", "github-profilegen-go")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, body)
		}

		var repos []Repository
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		// Check if we've received less than perPage repos, which means we've reached the last page
		if len(repos) < perPage {
			break
		}

		page++
	}

	return allRepos, nil
}

// shouldExcludeRepo determines if a repository should be excluded
func shouldExcludeRepo(repoName string, excludeList []string) bool {
	for _, name := range excludeList {
		if strings.EqualFold(repoName, name) {
			return true
		}
	}
	return false
}

// getPriorityIndex returns the priority index of a repository (-1 if not in priority list)
func getPriorityIndex(repoName string, priorityList []string) int {
	for i, name := range priorityList {
		if strings.EqualFold(repoName, name) {
			return i
		}
	}
	return -1
}

// generateReadme generates a README.md file with repository cards
func generateReadme(repos []Repository, config Config, contactInfo []string, aiCredits map[string]AICredit) error {
	// Define the template for repository cards
	const templateText = `# Repositories

<div style="display: flex; flex-wrap: wrap;">

{{range .Repos}}<!-- Repository: {{.Repository.Name}} -->
<div style="border: 1px solid #e1e4e8; border-radius: 6px; padding: 16px; margin: 8px; width: 320px;">
  <h3>
    📦 <a href="{{.Repository.HTMLURL}}">{{.Repository.Name}}</a>
  </h3>
  <p>{{if .Repository.Description}}{{.Repository.Description}}{{else}}No description provided{{end}}</p>
{{if .AICredit}}
<a href="#">
<img src="{{.AICredit.ImagePath}}" alt="{{.AICredit.AltText}}" title="{{.AICredit.TitleText}}" width="{{.AICredit.Width}}" height="{{.AICredit.Height}}">
</a>
{{end}}

  <p>
    {{if .Repository.Language}}🔵 {{.Repository.Language}}{{else}}📄 No language detected{{end}}  
    
      Created: {{.Repository.CreatedAt.Format "Jan 02, 2006"}}  
      Updated: {{.Repository.UpdatedAt.Format "Jan 02, 2006"}}  
    Published: {{.Repository.PushedAt.Format "Jan 02, 2006"}}  
  </p>

{{if .Repository.Fork}}🍴 Forked{{if .Repository.Source}}{{if .Repository.Source.HTMLURL}} from <a href="{{.Repository.Source.HTMLURL}}">source</a>{{end}}{{end}}  
{{end}}

</div>
{{end}}

</div>

{{if .ContactInfo}}
## Contact

{{range .ContactInfo}}
{{.}}
{{end}}
{{end}}
`

	// Prepare template data
	type TemplateRepo struct {
		Repository Repository
		AICredit   *AICredit
	}

	type TemplateData struct {
		Repos       []TemplateRepo
		ContactInfo []string
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
		Repos:       templateRepos,
		ContactInfo: contactInfo,
	}

	// Create and parse template
	tmpl, err := template.New("readme").Parse(templateText)
	if err != nil {
		return err
	}

	// Create output file
	file, err := os.Create(config.OutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	return nil
}

func main() {
	var showVersion bool
	// Parse command-line flags
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	username := flag.String("user", "", "GitHub username (required)")
	excludeFile := flag.String("exclude", "", "Path to exclusion list file")
	priorityFile := flag.String("priority", "", "Path to priority list file")
	contactFile := flag.String("contact", "", "Path to contact info file")
	aiCreditFile := flag.String("ai-credits", "", "Path to AI credits file")
	outputFile := flag.String("output", "README.md", "Path to output file")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s\n", os.Args[0], VERSION)
		os.Exit(0)
	}

	if *username == "" {
		fmt.Println("Error: GitHub username is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize configuration
	config := Config{
		Username:     *username,
		ExcludeFile:  *excludeFile,
		PriorityFile: *priorityFile,
		AICreditFile: *aiCreditFile,
		OutputFile:   *outputFile,
	}

	// Load exclusion list
	excludeList, err := loadTextFile(config.ExcludeFile)
	if err != nil {
		fmt.Printf("Error loading exclusion file: %v\n", err)
		os.Exit(1)
	}

	// Load priority list
	priorityList, err := loadTextFile(config.PriorityFile)
	if err != nil {
		fmt.Printf("Error loading priority file: %v\n", err)
		os.Exit(1)
	}

	// Load AI credits
	aiCredits, err := loadAICredits(config.AICreditFile)
	if err != nil {
		fmt.Printf("Error loading AI credits file: %v\n", err)
		os.Exit(1)
	}

	// Fetch repositories
	fmt.Printf("Fetching repositories for user %s...\n", config.Username)
	repos, err := fetchRepositories(config.Username)
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d repositories\n", len(repos))

	// Filter and sort repositories
	var filteredRepos []Repository
	for _, repo := range repos {
		if !shouldExcludeRepo(repo.Name, excludeList) {
			filteredRepos = append(filteredRepos, repo)
		}
	}
	fmt.Printf("After excluding, %d repositories remain\n", len(filteredRepos))

	// Sort repositories based on priority list and then by update time
	sort.Slice(filteredRepos, func(i, j int) bool {
		iPriority := getPriorityIndex(filteredRepos[i].Name, priorityList)
		jPriority := getPriorityIndex(filteredRepos[j].Name, priorityList)

		// If both are in priority list, sort by priority index
		if iPriority >= 0 && jPriority >= 0 {
			return iPriority < jPriority
		}

		// If only one is in priority list, it comes first
		if iPriority >= 0 {
			return true
		}
		if jPriority >= 0 {
			return false
		}

		// Otherwise, sort by update time (newest first)
		return filteredRepos[i].UpdatedAt.After(filteredRepos[j].UpdatedAt)
	})

	// Load contact information if provided
	var contactInfo []string
	if *contactFile != "" {
		contactInfo, err = loadTextFile(*contactFile)
		if err != nil {
			fmt.Printf("Error loading contact file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded contact information with %d lines\n", len(contactInfo))
	}

	// Generate README
	fmt.Printf("Generating README to %s...\n", config.OutputFile)
	if err := generateReadme(filteredRepos, config, contactInfo, aiCredits); err != nil {
		fmt.Printf("Error generating README: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
