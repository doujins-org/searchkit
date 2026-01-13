package migrations

import (
	"embed"
	"io/fs"
)

// migratekit's LoadFromFS does not recurse into subdirectories, so we expose a
// sub-filesystem rooted at "postgres/".
//
//go:embed postgres/*.sql
var postgresFS embed.FS

var Postgres fs.FS = mustSubFS(postgresFS, "postgres")

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
