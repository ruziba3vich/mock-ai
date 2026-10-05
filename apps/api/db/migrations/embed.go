package migrations

import "embed"

// Files contains the SQL migrations shipped with the API image.
//
//go:embed *.sql
var Files embed.FS
