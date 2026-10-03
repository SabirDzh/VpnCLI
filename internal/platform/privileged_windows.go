//go:build windows

package platform

import "golang.org/x/sys/windows"

// IsPrivileged reports whether the process runs elevated (admin),
// which is required to manage the TUN interface.
func IsPrivileged() bool {
	var adminSID *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY, 2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0, &adminSID)
	if err != nil {
		return false
	}
	defer windows.FreeSid(adminSID)

	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	member, err := token.IsMember(adminSID)
	return err == nil && member
}
