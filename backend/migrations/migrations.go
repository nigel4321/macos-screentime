// Package migrations bundles the SQL migration files as an embed.FS so the
// goose runner can execute them without depending on the working directory.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
