package worker

import "os"

// openForWrite opens a file for writing, creating parent dirs as needed.
func openForWrite(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}
