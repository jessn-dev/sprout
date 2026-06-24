package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectDocker(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{JavaVersion: "21", ArtifactId: "demo", BuildTool: "maven-project",
		Deps: []string{"data-redis", "data-neo4j"}}
	err = InjectDocker(tmpDir, cfg)
	if err != nil {
		t.Fatalf("InjectDocker failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "Dockerfile")); os.IsNotExist(err) {
		t.Errorf("Dockerfile was not created")
	}

	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	strContent := string(content)
	if len(strContent) == 0 {
		t.Errorf("docker-compose.yml is empty")
	}
	if !strings.Contains(strContent, "redis:") {
		t.Errorf("docker-compose.yml missing redis service")
	}
	if !strings.Contains(strContent, "neo4j:") {
		t.Errorf("docker-compose.yml missing neo4j service")
	}
}

func TestInjectJavaTemplates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "java-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{Language: "java", PackageName: "com.example.test",
		ProjectType: "standard", Deps: []string{"data-redis", "web"}}
	err = InjectSourceTemplates(tmpDir, cfg)
	if err != nil {
		t.Fatalf("InjectSourceTemplates failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "src", "main", "java", "com", "example", "test")
	if _, err := os.Stat(filepath.Join(expectedPath, "HelloController.java")); os.IsNotExist(err) {
		t.Errorf("HelloController.java was not created")
	}
	if _, err := os.Stat(filepath.Join(expectedPath, "RedisConfig.java")); os.IsNotExist(err) {
		t.Errorf("RedisConfig.java was not created")
	}
}
