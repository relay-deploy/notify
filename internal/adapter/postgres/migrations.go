package postgres

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var embedded embed.FS

var Migrations = mustSub(embedded, "migrations")

func mustSub(files fs.FS, directory string) fs.FS {
	sub, err := fs.Sub(files, directory)
	if err != nil {
		panic(err)
	}
	return sub
}
