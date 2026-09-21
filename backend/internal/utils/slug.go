package utils

import (
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify normalizes a skill/certification name into a stable comparison
// key: lowercase, non-alphanumeric collapsed to single hyphens, trimmed.
// This is what lets the deterministic matching engine compare
// "Go" == "go" == "GoLang"-ish variants consistently. It intentionally does
// NOT do fuzzy/semantic matching -- that's the AI-assisted layer's job.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// aliases maps common synonyms to a single canonical slug so "golang" and
// "go" are recognized as the same required skill. This list is intentionally
// small, explicit, and documented -- it is not a black box. Extend it as
// real-world resumes reveal more synonyms.
var skillAliases = map[string]string{
	"golang":       "go",
	"js":           "javascript",
	"ts":           "typescript",
	"reactjs":      "react",
	"react-js":     "react",
	"nodejs":       "node",
	"node-js":      "node",
	"postgres":     "postgresql",
	"k8s":          "kubernetes",
	"ml":           "machine-learning",
	"ai":           "artificial-intelligence",
	"mysql-db":     "mysql",
	"restful-api":  "rest-api",
	"restful-apis": "rest-api",
	"rest-apis":    "rest-api",
	"api-rest":     "rest-api",
}

// CanonicalSkillSlug applies Slugify and then resolves known aliases so
// matching is not defeated by superficial naming differences.
func CanonicalSkillSlug(s string) string {
	slug := Slugify(s)
	if canon, ok := skillAliases[slug]; ok {
		return canon
	}
	return slug
}
