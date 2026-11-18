package handlers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// UploadStore gives the dashboard export and import access to uploaded
// service icons and wallpapers (services.ImageStore). Images are written
// through saveValidatedImage, like an upload: the content type is sniffed and
// the file name is generated, never taken from the caller.
type UploadStore struct {
	preferencesRepo *repository.PreferencesRepository
}

func NewUploadStore(preferencesRepo *repository.PreferencesRepository) *UploadStore {
	return &UploadStore{preferencesRepo: preferencesRepo}
}

var _ services.ImageStore = (*UploadStore)(nil)

// imageRules returns where a kind of image lives and what it may be. Icons
// take the favicon types too: a fetched favicon is stored as an upload.
func imageRules(kind services.ImageKind) (dir string, maxSize int64, allowedTypes string) {
	if kind == services.ImageWallpaper {
		return WallpaperUploadDir, MaxWallpaperSize, AllowedMimeTypes
	}
	return UploadDir, MaxUploadSize, FaviconAllowedMimeTypes
}

// Check validates an image without storing it
func (s *UploadStore) Check(kind services.ImageKind, data []byte) error {
	_, maxSize, allowedTypes := imageRules(kind)
	_, err := validateImage(data, maxSize, allowedTypes)
	return err
}

// Save stores an image and returns what the database refers to it by: the
// file name of an icon, the URL path of a wallpaper. A wallpaper is safe from
// the sweep until release is called, which the caller does once the path is
// stored (or given up on).
func (s *UploadStore) Save(kind services.ImageKind, data []byte) (name string, release func(), err error) {
	dir, maxSize, allowedTypes := imageRules(kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create upload directory: %w", err)
	}
	release = func() {}
	if kind == services.ImageWallpaper {
		wallpaperSweep.RLock()
		release = sync.OnceFunc(wallpaperSweep.RUnlock)
	}
	filename, err := saveValidatedImage(bytes.NewReader(data), dir, maxSize, allowedTypes)
	if err != nil {
		release()
		return "", nil, err
	}
	if kind == services.ImageWallpaper {
		return wallpaperURLPrefix + filename, release, nil
	}
	return filename, release, nil
}

// Load reads a stored image by the name Save returned
func (s *UploadStore) Load(kind services.ImageKind, name string) ([]byte, error) {
	path, err := uploadPath(kind, name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Remove deletes a stored image; a missing file is fine
func (s *UploadStore) Remove(kind services.ImageKind, name string) {
	if path, err := uploadPath(kind, name); err == nil {
		_ = os.Remove(path)
	}
}

// PruneWallpapers deletes the wallpapers nothing points at anymore
func (s *UploadStore) PruneWallpapers(ctx context.Context) {
	PruneWallpapers(ctx, s.preferencesRepo)
}

// uploadPath maps a stored name to its file. Names come from the database,
// which the API lets users write, so anything but a plain file name is refused.
func uploadPath(kind services.ImageKind, name string) (string, error) {
	dir, _, _ := imageRules(kind)
	if kind == services.ImageWallpaper {
		if !localWallpaper.MatchString(name) {
			return "", fmt.Errorf("not an uploaded wallpaper: %q", name)
		}
		name = strings.TrimPrefix(name, wallpaperURLPrefix)
	}
	if name == "" || name == "." || name == ".." || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("not an uploaded file name: %q", name)
	}
	return filepath.Join(dir, name), nil
}
