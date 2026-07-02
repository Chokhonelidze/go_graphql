package migrate

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlserver"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) error {
    if databaseURL == "" {
        return fmt.Errorf("database url is empty")
    }
    fmt.Printf("Database URL: %s\n", databaseURL)

    migrationsSource := os.Getenv("MIGRATIONS_SOURCE")
    if migrationsSource == "" {
        migrationsSource = "file://./migrations/sql"
    }

    if localDir, ok := localMigrationDir(migrationsSource); ok {
        hasUpFiles, err := hasUpMigrations(localDir)
        if err != nil {
            return fmt.Errorf("failed checking migrations directory: %w", err)
        }
        if !hasUpFiles {
            fmt.Printf("No migration files found in %s; skipping SQL migrations\n", localDir)
            return nil
        }
    }

    fmt.Println("Running database migrations...")
    fmt.Printf("Migration source: %s\n", migrationsSource)

    m, err := migrate.New(
        migrationsSource,
        databaseURL,
    )
    if err != nil {
        return fmt.Errorf("migrate init failed: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate up failed: %w", err)
    }

    fmt.Println("Database migrations applied successfully")
    return nil
}

func localMigrationDir(source string) (string, bool) {
    const prefix = "file://"
    if !strings.HasPrefix(source, prefix) {
        return "", false
    }

    dir := strings.TrimPrefix(source, prefix)
    if dir == "" {
        return "", false
    }

    return filepath.Clean(dir), true
}

func hasUpMigrations(dir string) (bool, error) {
    pattern := filepath.Join(dir, "*.up.sql")
    files, err := filepath.Glob(pattern)
    if err != nil {
        return false, err
    }

    return len(files) > 0, nil
}