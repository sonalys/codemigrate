package adapter

import (
	"context"
	"fmt"
	"io"
	"io/fs"
)

type ScriptMigration struct {
	version int64
	script  string
}

// NewScriptMigrationFromString creates a new Migration from a given file.
func NewScriptMigrationFromFile(version int64, fileSystem fs.FS, path string) (*ScriptMigration, error) {
	if version <= 0 {
		return nil, fmt.Errorf("version must be greater than 0")
	}

	file, err := fileSystem.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration content: %w", err)
	}

	migration := &ScriptMigration{
		version: version,
		script:  string(content),
	}

	return migration, nil
}

// NewScriptMigrationFromReader creates a new Migration from a reader.
func NewScriptMigrationFromReader(version int64, reader io.Reader) (*ScriptMigration, error) {
	if version <= 0 {
		return nil, fmt.Errorf("version must be greater than 0")
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration content: %w", err)
	}

	if readCloser := reader.(io.ReadCloser); readCloser != nil {
		if err := readCloser.Close(); err != nil {
			return nil, fmt.Errorf("failed to close reader: %w", err)
		}
	}

	migration := &ScriptMigration{
		version: version,
		script:  string(content),
	}

	return migration, nil
}

func (m *ScriptMigration) Up(ctx context.Context, tx *Versioner) error {
	_, err := tx.Exec(ctx, m.script)
	if err != nil {
		return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
	}
	return nil
}

func (m *ScriptMigration) Down(ctx context.Context, tx *Versioner) error {
	_, err := tx.Exec(ctx, m.script)
	if err != nil {
		return fmt.Errorf("failed to revert migration %d: %w", m.version, err)
	}
	return nil
}

func (m *ScriptMigration) Version() int64 {
	return m.version
}
