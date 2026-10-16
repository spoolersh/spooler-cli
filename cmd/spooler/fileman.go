package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gobwas/cli"
)

// fileman opens the files a command writes and sees them closed: a file is
// opened once however many flags name it and through whatever symlinks, an
// existing file is replaced by its new content only when that content is
// complete, and a file the fileman created is removed when the command
// fails while it is still open, so a failure leaves no half-written file
// behind and no existing file lost for nothing. A file closed through Close
// is delivered and the caller's.
type fileman struct {
	// overwrite lets an existing regular file be replaced; without it the
	// file is refused with a usage error.
	overwrite bool

	// files maps each resolved, cleaned path to what Open made of it.
	files map[string]*dest
}

// dest is a destination Open made of a path.
type dest struct {
	file  *os.File
	state fileState

	// path is where a fileReplacing file is renamed on Close.
	path string
}

// fileState is what Open found at a path.
type fileState int

const (
	// fileStream is a stream, written as is and never removed.
	fileStream fileState = iota
	// fileCreated is a regular file Open created, removed on a failure.
	fileCreated
	// fileReplacing is a temporary file beside an existing regular file,
	// renamed over it on Close and removed when still open at Cleanup.
	fileReplacing
)

// Open opens name for writing, with perm for a file it creates.
//
// "-" is stdout. A stream, which is a path under /dev or /proc (/dev/stderr,
// /dev/fd/3) or an existing file that is not regular (a pipe), is written as
// is, appending. The file stdout is redirected to is stdout. Otherwise a
// missing file is created exclusively, and an existing one is, with
// overwrite, replaced on Close by a file created beside it, else refused
// with a usage error. A name already opened returns the same file.
func (m *fileman) Open(name string, perm os.FileMode) (*os.File, error) {
	if name == "-" {
		return os.Stdout, nil
	}
	// The destination is the file a symlink points at, so a replacement
	// lands there and the link stays; a stream is taken as spelled, since
	// what a descriptor resolves to is the shell's business.
	path := name
	if !isStreamPath(name) {
		path = resolve(name)
	}
	key := filepath.Clean(path)
	if d, has := m.files[key]; has {
		return d.file, nil
	}
	if d := m.openedAs(path); d != nil {
		return d.file, nil
	}
	// A replacement of the file stdout writes would unlink what stdout
	// wrote, so the path is stdout, with the record ahead of the payload.
	if isStdout(path) {
		return os.Stdout, nil
	}
	f, state, err := m.openDest(name, path, perm)
	if err != nil {
		return nil, err
	}
	if m.files == nil {
		m.files = make(map[string]*dest)
	}
	m.files[key] = &dest{
		file:  f,
		state: state,
		path:  path,
	}
	return f, nil
}

// resolve returns name made absolute with its symlinks resolved; when
// nothing is at name yet, its directory is resolved and the last element
// kept as spelled, so two spellings of one path make one key before the
// file exists.
func resolve(name string) string {
	if name == "" {
		// An empty name stays one, for the OS to refuse; it is not the
		// working directory.
		return name
	}
	if !filepath.IsAbs(name) {
		if wd, err := os.Getwd(); err == nil {
			// Not filepath.Abs or Join: they collapse ".." as spelled, and
			// "view/../bundle" with view a symlink is not "bundle".
			name = wd + string(filepath.Separator) + name
		}
	}
	if path, err := filepath.EvalSymlinks(name); err == nil {
		return path
	}
	dir, base := filepath.Split(name)
	if path, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(path, base)
	}
	return name
}

// openedAs returns the destination already opened for the file at path
// under another spelling that no resolution makes equal, as a
// case-insensitive filesystem has, judged by identity; nil when there is
// none, or nothing is at path yet.
func (m *fileman) openedAs(path string) *dest {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	for key, d := range m.files {
		if other, err := os.Stat(key); err == nil && os.SameFile(info, other) {
			return d
		}
	}
	return nil
}

// isStdout reports whether the file at path is the one stdout is open on,
// as a redirect makes it; false when nothing is at path.
func isStdout(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	out, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return os.SameFile(info, out)
}

// Close closes f when its content is complete, for a close error to count
// as a write error for the file's last bytes, and moves it over the path it
// replaces. A file this fileman created is removed on an error, and a file
// closed without one is the caller's, which Cleanup leaves alone. A file
// Open did not return, stdout included, is left alone too.
func (m *fileman) Close(f *os.File) error {
	key, d := m.lookup(f)
	if d == nil {
		return nil
	}
	delete(m.files, key)
	err := f.Close()
	if err == nil && d.state == fileReplacing {
		err = os.Rename(f.Name(), d.path)
	}
	if err != nil && d.state != fileStream {
		os.Remove(f.Name())
	}
	return err
}

// Cleanup closes every file still open and removes what must not stay: a
// replacing file, whose content was never completed, and with *err set a
// created one too. It is for a deferred call with the named error result.
func (m *fileman) Cleanup(err *error) {
	for _, d := range m.files {
		d.file.Close()
		if d.state == fileReplacing || (d.state == fileCreated && *err != nil) {
			os.Remove(d.file.Name())
		}
	}
}

// lookup finds the destination holding f, or nil.
func (m *fileman) lookup(f *os.File) (string, *dest) {
	for key, d := range m.files {
		if d.file == f {
			return key, d
		}
	}
	return "", nil
}

// openDest opens path for writing, as Open describes; name is the path as
// the caller spelled it, for messages.
func (m *fileman) openDest(name, path string, perm os.FileMode) (f *os.File, state fileState, err error) {
	if isStreamPath(path) {
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		return f, fileStream, err
	}
	f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err == nil {
		return f, fileCreated, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return nil, 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		return f, fileStream, err
	}
	if !m.overwrite {
		return nil, 0, cli.Exitf(2,
			"file exists: %s; pass --overwrite to replace it",
			name,
		)
	}
	// A rename replaces one name of a file; its other hard links would keep
	// the old content, so such a file is not replaced at all.
	if linkCount(info) > 1 {
		return nil, 0, cli.Exitf(2,
			"%s has other names (hard links): a replacement would update"+
				" only this one",
			name,
		)
	}
	f, err = createBeside(path, perm)
	if err != nil {
		return nil, 0, err
	}
	return f, fileReplacing, nil
}

// linkCount is the number of names info's file has; 1 when the system does
// not say.
func linkCount(info os.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink)
	}
	return 1
}

// createBeside creates a new file in name's directory with perm, under a
// name derived from it, for a rename over name once it is complete.
// os.CreateTemp is not used since it fixes the mode at 0600.
func createBeside(name string, perm os.FileMode) (*os.File, error) {
	dir, base := filepath.Split(name)
	for range 100 {
		tmp := filepath.Join(dir, fmt.Sprintf(".%s.%08x", base, rand.Uint32()))
		f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf(
		"create a file beside %s: too many attempts",
		name,
	)
}

// isStreamPath reports whether name addresses a stream rather than a file
// to create: the process's descriptors and devices under /dev, and their
// /proc spellings on Linux. What a redirect made of the descriptor, a file
// included, is the shell's business.
func isStreamPath(name string) bool {
	return strings.HasPrefix(name, "/dev/") || strings.HasPrefix(name, "/proc/")
}

// sameFile reports whether f0 and f1 are one file or stream, whatever paths
// opened them (-, /dev/stdout, /dev/fd/1 and a path all name stdout). A nil
// file is no file and matches nothing.
func sameFile(f0, f1 *os.File) (bool, error) {
	if f0 == nil || f1 == nil {
		return false, nil
	}
	i0, err := f0.Stat()
	if err != nil {
		return false, err
	}
	i1, err := f1.Stat()
	if err != nil {
		return false, err
	}
	return os.SameFile(i0, i1), nil
}

// fileTailWriter is a file that remembers the last two bytes written through
// it, to tell whether the output so far ends with an empty line.
type fileTailWriter struct {
	file *os.File
	tail [2]byte
}

func (w *fileTailWriter) Fd() uintptr {
	return w.file.Fd()
}

func (w *fileTailWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	switch {
	case n >= 2:
		w.tail = [2]byte{p[n-2], p[n-1]}
	case n == 1:
		w.tail = [2]byte{w.tail[1], p[0]}
	}
	return n, err
}

func (w *fileTailWriter) endsWithEmptyLine() bool {
	return w.tail == [2]byte{'\n', '\n'}
}
