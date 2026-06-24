package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/jessn-dev/sprout/internal/generator"
)

var quickCmd = &cobra.Command{
	Use:   "quick",
	Short: "Accelerated, streamlined project setup with sensible defaults",
	Run: func(cmd *cobra.Command, args []string) {
		var projectName string
		var profile string

		fmt.Println("\n🚀 Accelerated Setup")

		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Project Name").
					Placeholder("e.g. demo").
					Value(&projectName),

				huh.NewSelect[string]().
					Title("Starting Template").
					Options(
						huh.NewOption("Spring Boot (Standard)", "standard"),
						huh.NewOption("Spring Cloud (Microservices)", "cloud"),
						huh.NewOption("Spring Security (Secured Web App)", "security"),
					).
					Value(&profile),
			),
		).Run()

		if err != nil {
			log.Fatal(err)
		}

		projectName = strings.TrimSpace(projectName)
		if projectName == "" {
			projectName = "demo"
		}

		md, _ := generator.FetchMetadata()
		bootVersion := "" // empty → start.spring.io uses its current default
		if md != nil {
			bootVersion = generator.LatestGA(md) // prefer GA over milestones
		}

		cfg := generator.Config{
			BuildTool:   "maven-project",
			Language:    "java",
			BootVersion: bootVersion,
			GroupId:     "com.example",
			ArtifactId:  projectName,
			Name:        projectName,
			Description: "Spring Boot Application",
			PackageName: "com.example." + strings.ReplaceAll(projectName, "-", ""),
			Packaging:   "jar",
			JavaVersion: "21",
			ProjectType: profile,
			Deps:        []string{}, // Generate will append based on project type
		}

		if err := generator.ValidateConfig(cfg); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nGenerating %s...\n", projectName)
		if err := generator.Generate(cfg); err != nil {
			log.Fatalf("Failed to generate project: %v", err)
		}

		dir := generator.FolderName(cfg)
		fmt.Printf("✓ Project generated successfully at ./%s\n", dir)
		printNextSteps(dir)
	},
}

func init() {
	rootCmd.AddCommand(quickCmd)
}
