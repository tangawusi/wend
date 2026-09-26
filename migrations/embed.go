// Package migrations embeds the SQL migration files so the binary can
// apply them without shipping loose files alongside it.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
