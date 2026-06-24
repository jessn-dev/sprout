package generator

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	pkgPattern      = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)
	artifactPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
)

// ValidateConfig checks coordinates that start.spring.io would otherwise
// reject with an opaque 400. Returns nil when the config is submittable.
func ValidateConfig(cfg Config) error {
	var problems []string

	if strings.TrimSpace(cfg.ArtifactId) == "" {
		problems = append(problems, "Artifact ID is required")
	} else if !artifactPattern.MatchString(cfg.ArtifactId) {
		problems = append(problems, "Artifact ID may only contain letters, digits, '.', '-', '_' (no spaces)")
	}

	if strings.TrimSpace(cfg.GroupId) == "" {
		problems = append(problems, "Group ID is required")
	} else if !pkgPattern.MatchString(cfg.GroupId) {
		problems = append(problems, "Group ID must be a valid package (e.g. com.example)")
	}

	if strings.TrimSpace(cfg.PackageName) == "" {
		problems = append(problems, "Package Name is required")
	} else if !pkgPattern.MatchString(cfg.PackageName) {
		problems = append(problems, "Package Name must be a valid package (e.g. com.example.demo)")
	}

	if strings.TrimSpace(cfg.Name) == "" {
		problems = append(problems, "Project Name is required")
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
}
