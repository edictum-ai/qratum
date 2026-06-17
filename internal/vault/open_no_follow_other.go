//go:build !unix

package vault

import "os"

func openFileNoFollowRead(path string) (*os.File, error) {
	return os.Open(path)
}
