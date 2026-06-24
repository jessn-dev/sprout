package main

import (
	"log"
	"sprout/internal/generator"
)

func main() {
	cfg := generator.Config{
		BuildTool:   "gradle-project",
		Language:    "java",
		BootVersion: "3.3.0",
		GroupId:     "com.test",
		ArtifactId:  "demo-test",
		Name:        "demo-test",
		Description: "Test Project",
		PackageName: "com.test.demo",
		Packaging:   "jar",
		JavaVersion: "21",
		ProjectType: "standard",
		Deps:        []string{"web", "postgresql", "docker"},
	}
	err := generator.Generate(cfg)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
