package handlers

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// removeLocalAvatar deletes the on-disk avatar file referenced by avatarURL
// when it lives under /uploads/avatars/. No-op for nil pointers, remote URLs
// (OAuth providers), or files that are already gone. Errors are logged so
// orphans show up in ops logs, but never returned — callers can't usefully
// recover from a failed unlink.
func removeLocalAvatar(avatarURL *string, caller string) {
	if avatarURL == nil || !strings.HasPrefix(*avatarURL, "/uploads/avatars/") {
		return
	}
	// filepath.Base strips any directory components an attacker might have
	// smuggled into a stored URL.
	filename := filepath.Base(strings.TrimPrefix(*avatarURL, "/uploads/avatars/"))
	if filename == "." || filename == "/" || filename == "" {
		return
	}
	path := filepath.Join(AvatarUploadDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("%s: failed to remove avatar %q: %v", caller, path, err)
	}
}
