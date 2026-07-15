package migrations

import (
	"fmt"

	_ "embed"

	"gorm.io/gorm"
)

//go:embed 001_optimize_indexes.up.sql
var optimizeIndexesUp string

//go:embed 002_client_name_partial_unique.up.sql
var clientNamePartialUniqueUp string

//go:embed 003_rename_proxy_to_client_proxy.up.sql
var renameProxyToClientProxyUp string

func upSQLBeforeAutoMigrate() []string {
	return []string{renameProxyToClientProxyUp}
}

func upSQLAfterAutoMigrate() []string {
	return []string{
		optimizeIndexesUp,
		clientNamePartialUniqueUp,
	}
}

// UpSQL returns all migrations in logical execution order.
func UpSQL() []string {
	before := upSQLBeforeAutoMigrate()
	after := upSQLAfterAutoMigrate()
	all := make([]string, 0, len(before)+len(after))
	all = append(all, before...)
	all = append(all, after...)
	return all
}

// Apply runs SQL migrations for the given phase.
func Apply(db *gorm.DB, beforeAutoMigrate bool) error {
	var scripts []string
	if beforeAutoMigrate {
		scripts = upSQLBeforeAutoMigrate()
	} else {
		scripts = upSQLAfterAutoMigrate()
	}
	for _, sql := range scripts {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
