package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

const catalogDataPath = "data/catalog-data.sql"

var seedCatalogCmd = &cobra.Command{
	Use:   "seed-catalog",
	Short: "Replace prebuilt catalog data",
	RunE: func(cmd *cobra.Command, args []string) error {
		return seedCatalog(cmd.Context(), catalogDataPath)
	},
}

func init() {
	rootCmd.AddCommand(seedCatalogCmd)
}

func seedCatalog(ctx context.Context, path string) error {
	db, closeDB, err := openCommandDB()
	if err != nil {
		return err
	}
	defer closeDB()

	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read catalog seed file %s: %w", path, err)
	}

	seedSQL := stripPgDumpMetaCommands(string(sqlBytes))

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM prebuilt_meals; DELETE FROM meal_categories;").Error; err != nil {
			return fmt.Errorf("clear catalog tables: %w", err)
		}

		if err := tx.Exec(seedSQL).Error; err != nil {
			return fmt.Errorf("load catalog seed data: %w", err)
		}

		return nil
	})
}

func stripPgDumpMetaCommands(sqlText string) string {
	lines := strings.Split(sqlText, "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, `\`) {
			continue
		}

		if strings.EqualFold(trimmed, "SET transaction_timeout = 0;") {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
}
