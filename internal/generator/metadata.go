package generator

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type BootVersion struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Dependency is a single selectable Spring Initializr module.
type Dependency struct {
	ID           string
	Name         string
	Description  string
	Group        string
	VersionRange string // Maven-style range, e.g. "[3.1.0,3.4.0-M1)"; may be empty.
}

// Metadata is the slice of start.spring.io metadata that Sprout uses.
type Metadata struct {
	BootDefault  string
	BootVersions []BootVersion
	Dependencies []Dependency
}

type rawMetadata struct {
	BootVersion struct {
		Default string        `json:"default"`
		Values  []BootVersion `json:"values"`
	} `json:"bootVersion"`
	Dependencies struct {
		Values []struct {
			Name   string `json:"name"`
			Values []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Description  string `json:"description"`
				VersionRange string `json:"versionRange"`
			} `json:"values"`
		} `json:"values"`
	} `json:"dependencies"`
}

// FetchMetadata queries the Spring Initializr metadata endpoint for boot
// versions and the full dependency catalogue (including version ranges).
func FetchMetadata() (*Metadata, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://start.spring.io", nil)
	if err != nil {
		return nil, err
	}
	// v2.2 includes per-dependency versionRange.
	req.Header.Set("Accept", "application/vnd.initializr.v2.2+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw rawMetadata
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	md := &Metadata{
		BootDefault:  raw.BootVersion.Default,
		BootVersions: raw.BootVersion.Values,
	}
	for _, group := range raw.Dependencies.Values {
		for _, d := range group.Values {
			md.Dependencies = append(md.Dependencies, Dependency{
				ID:           d.ID,
				Name:         d.Name,
				Description:  d.Description,
				Group:        group.Name,
				VersionRange: d.VersionRange,
			})
		}
	}
	return md, nil
}

// isPrerelease reports whether a version carries a non-GA qualifier.
func isPrerelease(v string) bool {
	_, q := splitVersion(v)
	if q == "" {
		return false
	}
	u := strings.ToUpper(q)
	return u != "RELEASE" && u != "GA"
}

// LatestGA returns the newest non-prerelease Boot version, or BootDefault when
// none qualify. Avoids defaulting users onto milestone/RC builds.
func LatestGA(md *Metadata) string {
	best := ""
	for _, b := range md.BootVersions {
		if b.ID == "" || isPrerelease(b.ID) {
			continue
		}
		if best == "" || compareSpringVersion(b.ID, best) > 0 {
			best = b.ID
		}
	}
	if best == "" {
		return md.BootDefault
	}
	return best
}

// FallbackMetadata is a curated offline catalogue used when start.spring.io
// is unreachable, so the dependency picker still works.
func FallbackMetadata() *Metadata {
	curated := []struct{ id, name string }{
		{"web", "Spring Web"},
		{"security", "Spring Security"},
		{"data-jpa", "Spring Data JPA"},
		{"data-redis", "Redis Cache"},
		{"data-neo4j", "Neo4j"},
		{"amqp", "RabbitMQ"},
		{"kafka", "Apache Kafka"},
		{"postgresql", "PostgreSQL Driver"},
		{"mysql", "MySQL Driver"},
		{"h2", "H2 Database"},
		{"lombok", "Lombok"},
		{"validation", "Validation"},
		{"actuator", "Spring Boot Actuator"},
		{"devtools", "Spring Boot DevTools"},
		{"thymeleaf", "Thymeleaf"},
		{"docker", "Docker (Dockerfile & Compose)"},
	}
	// Empty ID → bootVersion is omitted from the request, so start.spring.io
	// applies its own current default instead of a stale pinned version.
	md := &Metadata{
		BootDefault: "",
		BootVersions: []BootVersion{
			{ID: "", Name: "Server Default"},
		},
	}
	for _, c := range curated {
		md.Dependencies = append(md.Dependencies, Dependency{ID: c.id, Name: c.name})
	}
	return md
}

// FetchBootVersions kept for backward compatibility.
func FetchBootVersions() ([]BootVersion, error) {
	md, err := FetchMetadata()
	if err != nil {
		return nil, err
	}
	return md.BootVersions, nil
}

// ---------- version range matching ----------

// VersionCompatible reports whether bootVersion satisfies the dependency's
// version range. An empty/unparseable range is treated as compatible.
func VersionCompatible(bootVersion, versionRange string) bool {
	versionRange = strings.TrimSpace(versionRange)
	if versionRange == "" {
		return true
	}

	lower, lowerInc, upper, upperInc, ok := parseRange(versionRange)
	if !ok {
		return true
	}

	if lower != "" {
		c := compareSpringVersion(bootVersion, lower)
		if c < 0 || (c == 0 && !lowerInc) {
			return false
		}
	}
	if upper != "" {
		c := compareSpringVersion(bootVersion, upper)
		if c > 0 || (c == 0 && !upperInc) {
			return false
		}
	}
	return true
}

// parseRange handles Maven range syntax plus a bare version (treated as >=).
func parseRange(s string) (lower string, lowerInc bool, upper string, upperInc bool, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false, "", false, false
	}

	first, last := s[0], s[len(s)-1]
	bracketed := (first == '[' || first == '(') && (last == ']' || last == ')')
	if !bracketed {
		// Bare version: minimum, inclusive, no upper bound.
		return s, true, "", false, true
	}

	lowerInc = first == '['
	upperInc = last == ']'
	inner := s[1 : len(s)-1]
	parts := strings.SplitN(inner, ",", 2)
	lower = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		upper = strings.TrimSpace(parts[1])
	}
	return lower, lowerInc, upper, upperInc, true
}

// compareSpringVersion compares two Spring-style versions.
// Returns -1, 0, or 1. Handles "3.3.0", "3.4.0-M1", "2.4.0.RELEASE".
func compareSpringVersion(a, b string) int {
	an, aq := splitVersion(a)
	bn, bq := splitVersion(b)

	for i := 0; i < len(an) || i < len(bn); i++ {
		var x, y int
		if i < len(an) {
			x = an[i]
		}
		if i < len(bn) {
			y = bn[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return compareQualifier(aq, bq)
}

// splitVersion returns numeric components and a trailing qualifier.
func splitVersion(v string) ([]int, string) {
	v = strings.TrimSpace(v)
	tokens := strings.FieldsFunc(v, func(r rune) bool { return r == '.' || r == '-' })
	var nums []int
	qual := ""
	for i, tok := range tokens {
		if n, err := strconv.Atoi(tok); err == nil {
			nums = append(nums, n)
			continue
		}
		qual = strings.Join(tokens[i:], "-")
		break
	}
	return nums, qual
}

// qualifierRank orders pre-release qualifiers below release.
// SNAPSHOT < Mx < RCx < RELEASE/GA/"".
func qualifierRank(q string) (rank int, num int) {
	q = strings.ToUpper(strings.TrimSpace(q))
	switch {
	case q == "" || q == "RELEASE" || q == "GA":
		return 3, 0
	case strings.HasPrefix(q, "RC"):
		return 2, trailingNum(q[2:])
	case strings.HasPrefix(q, "M"):
		return 1, trailingNum(q[1:])
	case strings.HasPrefix(q, "SNAPSHOT"):
		return 0, 0
	default:
		return 1, 0
	}
}

func trailingNum(s string) int {
	s = strings.TrimLeft(s, "-.")
	n, _ := strconv.Atoi(s)
	return n
}

func compareQualifier(a, b string) int {
	ra, na := qualifierRank(a)
	rb, nb := qualifierRank(b)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}
	if na != nb {
		if na < nb {
			return -1
		}
		return 1
	}
	return 0
}
