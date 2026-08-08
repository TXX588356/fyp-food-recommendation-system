package cmd

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/config"
	"fyp/food-rs/internal/database"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

const migrationsDir = "db/migrations"

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigrations(cmd.Context(), migrationsDir)
	},
}

var migrateAndSeedCmd = &cobra.Command{
	Use:   "migrate-and-seed",
	Short: "Run database migrations and seed catalog data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrations(cmd.Context(), migrationsDir); err != nil {
			return err
		}

		return seedCatalog(cmd.Context(), catalogDataPath)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd, migrateAndSeedCmd)
}

func openCommandDB() (*gorm.DB, func() error, error) {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	db, err := database.NewPostgresDB(cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	return db, sqlDB.Close, nil
}

func runMigrations(ctx context.Context, dir string) error {
	db, closeDB, err := openCommandDB()
	if err != nil {
		return err
	}
	defer closeDB()

	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("find migration files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", dir)
	}

	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			continue
		}

		fmt.Printf("Applying %s\n", file)
		if err := db.WithContext(ctx).Exec(sqlText).Error; err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}
	return nil
}
