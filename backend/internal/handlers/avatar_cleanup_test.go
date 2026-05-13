package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func ptr(s string) *string { return &s }

// withTempAvatarDir swaps AvatarUploadDir for the duration of the test and
// restores the original value via t.Cleanup. Returns the temp dir path.
func withTempAvatarDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	original := AvatarUploadDir
	AvatarUploadDir = tempDir
	t.Cleanup(func() { AvatarUploadDir = original })
	return tempDir
}

func TestRemoveLocalAvatar(t *testing.T) {
	t.Run("removes a local avatar file", func(t *testing.T) {
		dir := withTempAvatarDir(t)
		path := filepath.Join(dir, "avatar.jpg")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}

		removeLocalAvatar(ptr("/uploads/avatars/avatar.jpg"), "test")

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("file should be gone, stat err=%v", err)
		}
	})

	t.Run("no-op for nil URL", func(t *testing.T) {
		withTempAvatarDir(t)
		// Just must not panic.
		removeLocalAvatar(nil, "test")
	})

	t.Run("no-op for remote OAuth URL", func(t *testing.T) {
		dir := withTempAvatarDir(t)
		// Write a sentinel file in the avatars dir. If the helper wrongly
		// derives a filename from a remote URL, this is the only thing it
		// could plausibly hit.
		sentinel := filepath.Join(dir, "sentinel")
		if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
			t.Fatalf("seed sentinel: %v", err)
		}

		removeLocalAvatar(ptr("https://lh3.googleusercontent.com/a/example"), "test")

		if _, err := os.Stat(sentinel); err != nil {
			t.Errorf("sentinel should be untouched: %v", err)
		}
	})

	t.Run("no-op when the file is already gone", func(t *testing.T) {
		withTempAvatarDir(t)
		// File never existed — helper should swallow the not-exist error.
		removeLocalAvatar(ptr("/uploads/avatars/missing.jpg"), "test")
	})

	t.Run("strips path traversal segments", func(t *testing.T) {
		dir := withTempAvatarDir(t)
		// An attacker-controlled URL ending in ../../etc/passwd should resolve
		// to AvatarUploadDir/passwd via filepath.Base, never escape the dir.
		target := filepath.Join(dir, "passwd")
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			t.Fatalf("seed target: %v", err)
		}
		outside := filepath.Join(t.TempDir(), "outside")
		if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
			t.Fatalf("seed outside: %v", err)
		}

		removeLocalAvatar(ptr("/uploads/avatars/../../etc/passwd"), "test")

		if _, err := os.Stat(outside); err != nil {
			t.Errorf("file outside avatar dir must not be touched: %v", err)
		}
		// The in-dir "passwd" file is fair game — Base("../../etc/passwd") == "passwd".
		// We don't assert on its state; the point is that escape is prevented.
	})

	t.Run("ignores empty filename after prefix strip", func(t *testing.T) {
		withTempAvatarDir(t)
		// "/uploads/avatars/" with no filename — TrimPrefix yields "",
		// filepath.Base("") yields ".", which the guard catches.
		removeLocalAvatar(ptr("/uploads/avatars/"), "test")
	})
}
