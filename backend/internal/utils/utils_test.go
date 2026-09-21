package utils

import "testing"

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := HashPassword("SecurePass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "SecurePass123" {
		t.Error("hash must not equal the plaintext password")
	}
	if !CheckPassword(hash, "SecurePass123") {
		t.Error("CheckPassword should succeed with the correct password")
	}
	if CheckPassword(hash, "WrongPassword") {
		t.Error("CheckPassword should fail with an incorrect password")
	}
}

func TestHashPassword_DifferentHashesEachTime(t *testing.T) {
	h1, _ := HashPassword("SamePassword123")
	h2, _ := HashPassword("SamePassword123")
	if h1 == h2 {
		t.Error("bcrypt should produce different hashes for the same password due to random salting")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Go":             "go",
		"REST API":       "rest-api",
		"  Docker  ":     "docker",
		"Node.js":        "node-js",
		"C++":            "c",
		"MySQL Database": "mysql-database",
	}
	for input, expected := range cases {
		got := Slugify(input)
		if got != expected {
			t.Errorf("Slugify(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestCanonicalSkillSlug_ResolvesAliases(t *testing.T) {
	cases := map[string]string{
		"Golang":    "go",
		"golang":    "go",
		"Go":        "go",
		"ReactJS":   "react",
		"Node.js":   "node",
		"Postgres":  "postgresql",
		"REST APIs": "rest-api",
	}
	for input, expected := range cases {
		got := CanonicalSkillSlug(input)
		if got != expected {
			t.Errorf("CanonicalSkillSlug(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestCanonicalSkillSlug_GoAndGolangMatch(t *testing.T) {
	// This is the exact property the matching engine depends on: two
	// different spellings of the same skill must normalize identically.
	if CanonicalSkillSlug("Go") != CanonicalSkillSlug("Golang") {
		t.Error("'Go' and 'Golang' must resolve to the same canonical slug")
	}
}
