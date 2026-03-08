package session

import "os"

var writeFileFunc = writeFileAtomically

func initializeSessionDirectories(sessionDir string) error {
	for _, item := range requiredDirectories {
		if err := os.MkdirAll(sessionJoin(sessionDir, item), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func writeSessionFiles(files []renderedFile) error {
	for _, item := range files {
		if err := writeFileFunc(item.Path, item.Content); err != nil {
			return err
		}
	}
	return nil
}

func writeFileAtomically(path string, content []byte) error {
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}
