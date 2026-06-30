package migrations

import _ "embed"

//go:embed 001_optimize_indexes.up.sql
var optimizeIndexesUp string

//go:embed 002_client_name_partial_unique.up.sql
var clientNamePartialUniqueUp string

// UpSQL returns index migrations in execution order.
func UpSQL() []string {
	return []string{
		optimizeIndexesUp,
		clientNamePartialUniqueUp,
	}
}
