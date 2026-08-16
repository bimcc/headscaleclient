package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomic(destination string, data []byte, mode os.FileMode) (returnErr error) {
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := temporary.Close(); returnErr == nil && closeErr != nil {
				returnErr = closeErr
			}
		}
		if removeErr := os.Remove(temporaryPath); returnErr == nil && removeErr != nil && !os.IsNotExist(removeErr) {
			returnErr = removeErr
		}
	}()

	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	closed = true
	if err := replaceFile(temporaryPath, destination); err != nil {
		return fmt.Errorf("replace destination: %w", err)
	}
	return syncParentDirectory(directory)
}
