package adapter

import (
	"context"
	"fmt"
	"io"
	"io/fs"
)

type ScriptMigration[T Transaction] struct {
	version    int64
	upScript   string
	downScript string
}

// NewScriptMigrationFromString creates a new Migration from a given file.
func NewScriptMigrationFromFile[T Transaction](
	version int64,
	fileSystem fs.FS,
	upScriptPath string,
	downScriptPath string,
) (*ScriptMigration[T], error) {
	if version <= 0 {
		return nil, fmt.Errorf("version must be greater than 0")
	}

	upScript, err := readFileContent(fileSystem, upScriptPath)
	if err != nil {
		return nil, err
	}

	downScript, err := readFileContent(fileSystem, downScriptPath)
	if err != nil {
		return nil, err
	}

	migration := &ScriptMigration[T]{
		version:    version,
		upScript:   upScript,
		downScript: downScript,
	}

	return migration, nil
}

func readFileContent(fileSystem fs.FS, filePath string) (string, error) {
	file, err := fileSystem.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	return string(content), nil
}

// NewScriptMigrationFromReader creates a new Migration from readers for up and down scripts.
func NewScriptMigrationFromReader[T Transaction](
	version int64,
	upReader io.Reader,
	downReader io.Reader,
) (*ScriptMigration[T], error) {
	if version <= 0 {
		return nil, fmt.Errorf("version must be greater than 0")
	}

	upContent, err := safeReadAll(upReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read up migration content: %w", err)
	}
	if err := closeReader(upReader); err != nil {
		return nil, err
	}

	downContent, err := safeReadAll(downReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read down migration content: %w", err)
	}
	if err := closeReader(downReader); err != nil {
		return nil, err
	}

	migration := &ScriptMigration[T]{
		version:    version,
		upScript:   string(upContent),
		downScript: string(downContent),
	}

	return migration, nil
}

func safeReadAll(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return []byte{}, nil
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return []byte{}, nil
	}

	return content, nil
}

func closeReader(reader io.Reader) error {
	if readCloser, ok := reader.(io.ReadCloser); ok {
		if err := readCloser.Close(); err != nil {
			return fmt.Errorf("failed to close reader: %w", err)
		}
	}
	return nil
}

func (m *ScriptMigration[T]) Up(ctx context.Context, tx *Versioner[T]) error {
	_, err := tx.Tx.Exec(m.upScript)
	if err != nil {
		return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
	}
	return nil
}

func (m *ScriptMigration[T]) Down(ctx context.Context, tx *Versioner[T]) error {
	_, err := tx.Tx.Exec(m.downScript)
	if err != nil {
		return fmt.Errorf("failed to revert migration %d: %w", m.version, err)
	}
	return nil
}

func (m *ScriptMigration[T]) Version() int64 {
	return m.version
}
