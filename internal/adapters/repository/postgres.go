package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDB es un ayudante para conectar (opcional, si queremos limpiar el main después)
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
