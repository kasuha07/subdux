//go:build !unix

package pkg

func prepareDataPathRuntimeOwnership(string) error {
	return nil
}
