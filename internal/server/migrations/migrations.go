package migrations

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

const migrationsTable = "schema_migrations"

//go:embed *.sql
var migrationsFS embed.FS

func Auto(db *gorm.DB) error {

	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	versionBefore, _, err := versionOrZero(m)
	if err != nil {
		return fmt.Errorf("check migration version: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return downTo(m, versionBefore, err)
	}
	return nil

}

func Rollback(db *gorm.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

func newMigrator(db *gorm.DB) (*migrate.Migrate, error) {

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return nil, fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil

}

func versionOrZero(m *migrate.Migrate) (uint, bool, error) {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return version, dirty, nil
}

func downTo(m *migrate.Migrate, versionBefore uint, err error) error {
	versionAfter, dirty, vaErr := versionOrZero(m)
	if vaErr != nil {
		return fmt.Errorf("migration failed: %w (version check failed: %v)", err, vaErr)
	}
	if dirty {
		if forceErr := m.Force(int(versionAfter)); forceErr != nil {
			return fmt.Errorf("migration failed: %w (force dirty version failed: %v)", err, forceErr)
		}
		if rbErr := m.Steps(-1); rbErr != nil && !errors.Is(rbErr, migrate.ErrNoChange) {
			return fmt.Errorf("migration failed: %w (rollback dirty migration failed: %v)", err, rbErr)
		}
		return fmt.Errorf("migration failed: %w (rolled back dirty migration v%d)", err, versionAfter)
	}
	if versionAfter > versionBefore {
		if rbErr := m.Steps(-1); rbErr != nil && !errors.Is(rbErr, migrate.ErrNoChange) {
			return fmt.Errorf("migration failed: %w (rollback failed: %v)", err, rbErr)
		}
		return fmt.Errorf("migration failed: %w (rolled back to previous version v%d)", err, versionBefore)
	}
	return fmt.Errorf("migration failed: %w", err)
}
