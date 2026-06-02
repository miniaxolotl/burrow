package internal

import (
	"embed"
	"io/fs"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func MigrationsSource() (source.Driver, error) {
	fs, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	return iofs.New(fs, "migrations")
}