//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/database"
	"io"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var integrationDB *gorm.DB

func TestIntegration(t *testing.T) {
	silenceTestLogs()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func silenceTestLogs() {
	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

var _ = BeforeSuite(func() {
	dbURL := os.Getenv("DATABASE_URL")
	Expect(dbURL).NotTo(BeEmpty(), "DATABASE_URL is required for integration tests")

	guardTestDatabase(dbURL)

	db, err := database.NewPostgresDB(dbURL)
	Expect(err).NotTo(HaveOccurred())

	integrationDB = db
	applyMigrations(integrationDB)
})

var _ = AfterSuite(func() {
	if integrationDB == nil {
		return
	}

	sqlDB, err := integrationDB.DB()
	Expect(err).NotTo(HaveOccurred())
	Expect(sqlDB.Close()).To(Succeed())
})

func beginIntegrationTx() (*gorm.DB, context.Context) {
	tx := integrationDB.Begin()
	Expect(tx.Error).NotTo(HaveOccurred())

	DeferCleanup(func() {
		Expect(tx.Rollback().Error).NotTo(HaveOccurred())
	})

	return tx, context.Background()
}

func applyMigrations(db *gorm.DB) {
	files, err := filepath.Glob("../../db/migrations/*.up.sql")
	Expect(err).NotTo(HaveOccurred())
	Expect(files).NotTo(BeEmpty(), "no migration files found")

	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		Expect(err).NotTo(HaveOccurred())

		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			continue
		}

		err = db.Exec(sqlText).Error
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("apply migration %s", file))
	}
}

func guardTestDatabase(dbURL string) {
	parsed, err := url.Parse(dbURL)
	Expect(err).NotTo(HaveOccurred())

	dbName := strings.TrimPrefix(parsed.Path, "/")
	Expect(dbName).To(
		ContainSubstring("test"),
		"integration tests must use a database name containing 'test'",
	)
}
