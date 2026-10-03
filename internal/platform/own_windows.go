//go:build windows

package platform

// ChownToSudoUser is a no-op on Windows.
func ChownToSudoUser(_ string) {}
