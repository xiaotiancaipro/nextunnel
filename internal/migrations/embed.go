package migrations

import _ "embed"

//go:embed 001_optimize_indexes.up.sql
var OptimizeIndexesUp string
