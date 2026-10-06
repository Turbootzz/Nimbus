package handlers

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// removeLocalAvatar deletes the on-disk avatar file referenced by avatarURL
// when it lives under /uploads/avatars/. No-op for nil pointers, remote URLs
// (OAuth providers), or files that are already gone.
func removeLocalAvatar(avatarURL *string, caller string) {
	removeLocalUpload(avatarURL, "/uploads/avatars/", AvatarUploadDir, caller)
}

// removeLocalWallpaper deletes an uploaded wallpaper; remote URLs are left alone
func removeLocalWallpaper(wallpaperURL *string, caller string) {
	removeLocalUpload(wallpaperURL, wallpaperURLPrefix, WallpaperUploadDir, caller)
}

// removeLocalUpload deletes the file in dir that fileURL points at when it
// starts with urlPrefix. Errors are logged so orphans show up in ops logs,
// but never returned — callers can't usefully recover from a failed unlink.
func removeLocalUpload(fileURL *string, urlPrefix, dir, caller string) {
	if fileURL == nil || !strings.HasPrefix(*fileURL, urlPrefix) {
		return
	}
	// filepath.Base strips any directory components an attacker might have
	// smuggled into a stored URL.
	filename := filepath.Base(strings.TrimPrefix(*fileURL, urlPrefix))
	if filename == "." || filename == "/" || filename == "" {
		return
	}
	path := filepath.Join(dir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("%s: failed to remove %q: %v", caller, path, err)
	}
}
