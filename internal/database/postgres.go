package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  databaseURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
}
