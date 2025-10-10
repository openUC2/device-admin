package deviceadmin

import (
	"cmp"
	"io/fs"
	"iter"
	"maps"
	"path"
	"slices"

	"github.com/pkg/errors"
)

// OverlayFS

// A OverlayFS is an FS constructed by combining a [fs.FS] as an overlay over another [fs.FS] as an
// underlay.
type OverlayFS struct {
	Upper fs.FS
	Lower fs.FS
}

// OverlayFS: fs.FS

// Open opens the named file from the upper (if it exists in the upper), or else from the lower.
func (f *OverlayFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	// fmt.Printf("Open(%s|%s)\n", f.Path(), name)
	if f.Upper == nil {
		return f.Lower.Open(name)
	}

	file, err := f.Upper.Open(name)
	switch {
	default:
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't open file %s in upper", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		file, err = f.Lower.Open(name)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err:  errors.Wrapf(err, "couldn't open file %s in lower", name),
			}
		}
		return file, nil
	case err == nil:
		return file, nil
	}
}

// Sub returns a OverlayFS corresponding to the subtree rooted at dir.
func (f *OverlayFS) Sub(dir string) (*OverlayFS, error) {
	dir = path.Clean(dir)
	// fmt.Printf("Sub(%s|%s)\n", f.Path(), dir)
	if dir == "." {
		return f, nil
	}
	upperSub, err := fs.Sub(f.Upper, dir)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make subtree for upper")
	}
	lowerSub, err := fs.Sub(f.Lower, dir)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make subtree for lower")
	}

	return &OverlayFS{
		Upper: upperSub,
		Lower: lowerSub,
	}, nil
}

// OverlayFS: fs.ReadDirFS

// ReadDir reads the named directory and returns a list of directory entries sorted by filename.
func (f *OverlayFS) ReadDir(name string) (entries []fs.DirEntry, err error) {
	name = path.Clean(name)
	// fmt.Printf("ReadDir(%s|%s)\n", f.Path(), name)
	if f.Upper == nil {
		return fs.ReadDir(f.Lower, name)
	}

	entryNames := make(Set[string])

	info, err := fs.Stat(f.Upper, name)
	if err == nil {
		if !info.IsDir() {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "%s is a non-directory file in upper", name),
			}
		}
		if entries, err = fs.ReadDir(f.Upper, name); err != nil {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "couldn't read directory %s in upper", name),
			}
		}
		for _, entry := range entries {
			entryNames.Add(entry.Name())
		}
	}

	info, err = fs.Stat(f.Lower, name)
	if err == nil {
		if !info.IsDir() {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "%s is a non-directory file in lower", name),
			}
		}
		lowerEntries, err := fs.ReadDir(f.Lower, name)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "couldn't read directory %s in lower", name),
			}
		}
		for _, entry := range lowerEntries {
			if !entryNames.Has(entry.Name()) {
				entries = append(entries, entry)
				entryNames.Add(entry.Name())
			}
		}
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// OverlayFS: fs.ReadFileFS

// ReadFile returns the contents from reading the named file from the upper (if it exists in the
// upper), or else from the lower.
func (f *OverlayFS) ReadFile(name string) ([]byte, error) {
	name = path.Clean(name)
	// fmt.Printf("ReadFile(%s|%s)\n", f.Path(), name)
	if f.Upper == nil {
		return fs.ReadFile(f.Lower, name)
	}

	contents, err := fs.ReadFile(f.Upper, name)
	switch {
	default:
		return nil, errors.Wrapf(err, "couldn't read file %s in upper", name)
	case errors.Is(err, fs.ErrNotExist):
		contents, err := fs.ReadFile(f.Lower, name)
		return contents, errors.Wrapf(err, "couldn't read file %s in lower", name)
	case err == nil:
		return contents, nil
	}
}

// OverlayFS: fs.StatFS

// Stat returns a [fs.FileInfo] describing the file from the upper (if it exists in the upper),
// or else from the lower.
func (f *OverlayFS) Stat(name string) (fs.FileInfo, error) {
	name = path.Clean(name)
	// fmt.Printf("Stat(%s|%s)\n", f.Path(), name)
	if f.Upper == nil {
		return fs.Stat(f.Lower, name)
	}

	info, err := fs.Stat(f.Upper, name)
	switch {
	default:
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't stat file %s in upper", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		info, err := fs.Stat(f.Lower, name)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "stat",
				Path: name,
				Err:  errors.Wrapf(err, "couldn't stat file %s in lower", name),
			}
		}
		return info, nil
	case err == nil:
		return info, nil
	}
}

// Set

type Set[Node comparable] map[Node]struct{}

// Add adds the node to the set. If the node was already in the set, nothing changes.
func (s Set[Node]) Add(n ...Node) {
	for _, node := range n {
		s[node] = struct{}{}
	}
}

// Has checks whether the node is already in the set.
func (s Set[Node]) Has(n Node) bool {
	_, ok := s[n]
	return ok
}

// Difference creates a new set with the difference between the set whose method is called and the
// provided set.
func (s Set[Node]) Difference(t Set[Node]) Set[Node] {
	difference := make(Set[Node])
	for node := range s {
		if !t.Has(node) {
			difference.Add(node)
		}
	}
	return difference
}

// All returns an iterator over all elements in s.
func (s Set[Node]) All() iter.Seq[Node] {
	return maps.Keys(s)
}
