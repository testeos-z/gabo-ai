package adapters

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

const maxTreeBytes = 4 << 30

// DirWorkspace implements review.Workspace under a private root directory.
// Extraction goes through os.Root, so archive entries cannot escape it.
type DirWorkspace struct {
	root string
}

var _ review.Workspace = DirWorkspace{}

// NewDirWorkspace stores workspaces under root (created 0700).
func NewDirWorkspace(root string) DirWorkspace { return DirWorkspace{root: root} }

func (w DirWorkspace) path(name string) (string, error) {
	if err := review.ValidateWorkspace(name); err != nil {
		return "", err
	}
	return filepath.Join(w.root, name), nil
}

// Create replaces the workspace with the content of the tar stream r.
func (w DirWorkspace) Create(name string, r io.Reader) (string, error) {
	dir, err := w.path(name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(w.root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(w.root, 0o700); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	if err := extract(dir, r); err != nil {
		return "", errors.Join(err, os.RemoveAll(dir))
	}
	return dir, nil
}

// Dir returns the path of an existing workspace.
func (w DirWorkspace) Dir(name string) (string, error) {
	dir, err := w.path(name)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return dir, nil
}

// Home replaces the workspace's HOME, a sibling of its tree, with a fresh
// empty 0700 directory.
func (w DirWorkspace) Home(name string) (string, error) {
	dir, err := w.path(name)
	if err != nil {
		return "", err
	}
	home := dir + ".home"
	if err := os.MkdirAll(w.root, 0o700); err != nil {
		return "", err
	}
	if err := os.RemoveAll(home); err != nil {
		return "", err
	}
	if err := os.Mkdir(home, 0o700); err != nil {
		return "", err
	}
	return home, nil
}

// Remove deletes the workspace and its HOME; missing ones are not an error.
func (w DirWorkspace) Remove(name string) error {
	dir, err := w.path(name)
	if err != nil {
		return err
	}
	return errors.Join(os.RemoveAll(dir), os.RemoveAll(dir+".home"))
}

func extract(dir string, r io.Reader) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	tr := tar.NewReader(io.LimitReader(r, maxTreeBytes))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tree: %w", err)
		}
		name := path.Clean(h.Name)
		if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("unsafe path %q in tree", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(name, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(root, name, h, tr); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := mkParent(root, name); err != nil {
				return err
			}
			if err := root.Symlink(h.Linkname, name); err != nil {
				return err
			}
		case tar.TypeXGlobalHeader:
		default:
			return fmt.Errorf("unsupported entry %q of type %q", h.Name, h.Typeflag)
		}
	}
}

func writeFile(root *os.Root, name string, h *tar.Header, r io.Reader) error {
	if err := mkParent(root, name); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if h.FileInfo().Mode()&0o111 != 0 {
		mode = 0o700
	}
	f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		return errors.Join(err, f.Close())
	}
	return f.Close()
}

func mkParent(root *os.Root, name string) error {
	if dir := path.Dir(name); dir != "." {
		return root.MkdirAll(dir, 0o700)
	}
	return nil
}
