package migrate

type StringError string

const (
	// ErrNoMigrations when no migrations were applied.
	ErrNoMigrations = StringError("no migrations applied")
	// ErrMigrationNotFound when a migration is not found.
	ErrMigrationNotFound = StringError("migration not found")
	// ErrDuplicateMigration when a version is duplicated.
	ErrDuplicateMigration = StringError("duplicate migration version")
)

var (
	_ error = StringError("")
)

func (e StringError) Error() string {
	return string(e)
}
