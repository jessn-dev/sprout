package generator

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// startSpinner prints an animated spinner until the returned stop func runs.
func startSpinner(msg string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	go func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				fmt.Printf("\r\033[K✓ %s\n", msg)
				return
			default:
				fmt.Printf("\r%s %s", frames[i%len(frames)], msg)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return func() { close(done); time.Sleep(20 * time.Millisecond) }
}

var nameSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// sanitizeName makes a filesystem- and Spring-safe folder name.
func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = nameSanitizer.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// FolderName returns the destination directory Generate will create.
func FolderName(cfg Config) string {
	d := sanitizeName(cfg.ArtifactId)
	if d == "" {
		d = sanitizeName(cfg.Name)
	}
	if d == "" {
		d = "demo"
	}
	return d
}

// hasWeb reports whether the resulting project includes Spring Web, either via
// the chosen profile or an explicit dependency.
func hasWeb(cfg Config) bool {
	switch cfg.ProjectType {
	case "standard", "cloud", "security":
		return true
	}
	for _, d := range cfg.Deps {
		if d == "web" {
			return true
		}
	}
	return false
}

type Config struct {
	BuildTool   string
	Language    string
	BootVersion string
	GroupId     string
	ArtifactId  string
	Name        string
	Description string
	PackageName string
	Packaging   string
	JavaVersion string
	ProjectType string
	Deps        []string
}

func Generate(cfg Config) error {
	// 1. Build the dependencies string
	var springDeps []string
	switch cfg.ProjectType {
	case "standard":
		springDeps = append(springDeps, "web")
	case "cloud":
		springDeps = append(springDeps, "web", "cloud-gateway")
	case "security":
		springDeps = append(springDeps, "web", "security")
	}

	for _, d := range cfg.Deps {
		if d != "docker" {
			springDeps = append(springDeps, d)
		}
	}

	// 2. Construct the URL
	baseURL, _ := url.Parse("https://start.spring.io/starter.zip")
	params := url.Values{}
	params.Add("type", cfg.BuildTool)
	params.Add("language", cfg.Language)
	// Omit when unknown so start.spring.io uses its own current default
	// (avoids 400 from a stale pinned version).
	if cfg.BootVersion != "" {
		params.Add("bootVersion", cfg.BootVersion)
	}
	params.Add("groupId", cfg.GroupId)
	params.Add("artifactId", cfg.ArtifactId)
	params.Add("name", cfg.Name)
	params.Add("description", cfg.Description)
	params.Add("packageName", cfg.PackageName)
	params.Add("packaging", cfg.Packaging)
	params.Add("javaVersion", cfg.JavaVersion)
	params.Add("dependencies", strings.Join(springDeps, ","))
	baseURL.RawQuery = params.Encode()

	// Resolve a safe destination folder and guard against clobbering.
	destDir := FolderName(cfg)
	if _, statErr := os.Stat(destDir); statErr == nil {
		return fmt.Errorf("directory ./%s already exists; remove it or choose another name", destDir)
	}

	// 3. Make the HTTP Request (bounded so a network stall can't hang the CLI).
	client := &http.Client{Timeout: 30 * time.Second}
	stop := startSpinner("Contacting start.spring.io...")
	resp, err := client.Get(baseURL.String())
	stop()
	if err != nil {
		return fmt.Errorf("failed to reach Spring Initializr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Spring Initializr returned an error: %s", resp.Status)
	}

	// 4. Save the zip file
	zipName := destDir + "_project.zip"
	out, err := os.Create(zipName)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return fmt.Errorf("failed to save zip file: %w", err)
	}

	fmt.Println("✅ Downloaded project successfully!")

	fmt.Println("📦 Unzipping project...")
	err = Unzip(zipName, destDir)
	if err != nil {
		return fmt.Errorf("failed to unzip project: %w", err)
	}

	// Clean up the zip file
	os.Remove(zipName)

	// Inject custom templates
	for _, d := range cfg.Deps {
		if d == "docker" {
			fmt.Println("🐳 Injecting multi-stage Dockerfile and wired docker-compose.yml...")
			err = InjectDocker(destDir, cfg)
			if err != nil {
				log.Printf("Warning: Failed to inject docker templates: %v\n", err)
			}
		}
	}

	fmt.Println("☕ Injecting custom Java configs and controllers...")
	err = InjectJavaTemplates(destDir, cfg.PackageName, cfg.Language, cfg.Deps, hasWeb(cfg))
	if err != nil {
		log.Printf("Warning: Failed to inject Java templates: %v\n", err)
	}

	fmt.Println("⚙️  Wiring application.properties...")
	if err = InjectProperties(destDir, cfg); err != nil {
		log.Printf("Warning: Failed to inject properties: %v\n", err)
	}

	fmt.Printf("✅ Project unzipped to ./%s!\n", destDir)
	fmt.Printf("🚀 Sprout generation complete! Run 'cd %s' to get started.\n", destDir)

	return nil
}
