// Package migrations bundles the SQL migration files as an embed.FS so the
// goose runner can execute them without depending on the working directory.
package migrations

import "embed"

// FS is the embedded SQL migration tree consumed by goose.
//
//go:embed *.sql
var FS embed.FS
