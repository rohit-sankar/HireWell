package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// migrationFile represents one discovered *.up.sql / *.down.sql pair.
type migrationFile struct {
	version  string // e.g. "001_initial_schema"
	upPath   string
	downPath string
}

func discoverMigrations(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	byVersion := map[string]*migrationFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			v := strings.TrimSuffix(name, ".up.sql")
			m := byVersion[v]
			if m == nil {
				m = &migrationFile{version: v}
				byVersion[v] = m
			}
			m.upPath = filepath.Join(dir, name)
		case strings.HasSuffix(name, ".down.sql"):
			v := strings.TrimSuffix(name, ".down.sql")
			m := byVersion[v]
			if m == nil {
				m = &migrationFile{version: v}
				byVersion[v] = m
			}
			m.downPath = filepath.Join(dir, name)
		}
	}

	out := make([]migrationFile, 0, len(byVersion))
	for _, m := range byVersion {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(50) NOT NULL PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func appliedVersions(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// MigrateUp applies every *.up.sql migration in dir that has not yet been
// recorded in schema_migrations, in filename order.
func MigrateUp(db *sql.DB, dir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}
	migrations, err := discoverMigrations(dir)
	if err != nil {
		return err
	}
	applied, err := appliedVersions(db)
	if err != nil {
		return fmt.Errorf("load applied versions: %w", err)
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if m.upPath == "" {
			return fmt.Errorf("migration %s has no .up.sql file", m.version)
		}
		sqlBytes, err := os.ReadFile(m.upPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.upPath, err)
		}
		if err := execMultiStatement(db, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply %s: %w", m.version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			return fmt.Errorf("record %s: %w", m.version, err)
		}
		fmt.Printf("migrated up: %s\n", m.version)
	}
	return nil
}

// MigrateDown reverts the most recently applied migration.
func MigrateDown(db *sql.DB, dir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	migrations, err := discoverMigrations(dir)
	if err != nil {
		return err
	}
	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}

	// find the highest-versioned applied migration
	var target *migrationFile
	for i := len(migrations) - 1; i >= 0; i-- {
		if applied[migrations[i].version] {
			target = &migrations[i]
			break
		}
	}
	if target == nil {
		fmt.Println("no migrations to roll back")
		return nil
	}
	if target.downPath == "" {
		return fmt.Errorf("migration %s has no .down.sql file", target.version)
	}
	sqlBytes, err := os.ReadFile(target.downPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", target.downPath, err)
	}
	if err := execMultiStatement(db, string(sqlBytes)); err != nil {
		return fmt.Errorf("revert %s: %w", target.version, err)
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version = ?`, target.version); err != nil {
		return err
	}
	fmt.Printf("migrated down: %s\n", target.version)
	return nil
}

// execMultiStatement runs a .sql file that may contain multiple ';'-terminated
// statements. The mysql driver's multiStatements option would also work, but
// splitting explicitly keeps error messages attributable to a single statement.
func execMultiStatement(db *sql.DB, script string) error {
	statements := splitSQLStatements(script)
	for _, stmt := range statements {
		s := strings.TrimSpace(stmt)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec statement failed: %s: %w", truncate(s, 120), err)
		}
	}
	return nil
}

func splitSQLStatements(script string) []string {
	// Strip full-line comments, then split on ';'. Our migration files do not
	// use ';' inside string literals or DELIMITER blocks, so this simple
	// approach is correct for this schema.
	lines := strings.Split(script, "\n")
	var cleaned []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned = append(cleaned, l)
	}
	joined := strings.Join(cleaned, "\n")
	return strings.Split(joined, ";")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
