package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTailWriter(t *testing.T) {
	for _, test := range []struct {
		name   string
		writes []string
		exp    bool
	}{
		{
			name:   "record ending with an empty line",
			writes: []string{"id: 1\n\n"},
			exp:    true,
		},
		{
			name:   "empty line across writes",
			writes: []string{"id: 1\n", "\n"},
			exp:    true,
		},
		{
			name:   "one line only",
			writes: []string{"{\"id\":\"1\"}\n"},
		},
		{
			name: "nothing written",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.Create(filepath.Join(t.TempDir(), "out"))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			defer f.Close()
			w := &fileTailWriter{
				file: f,
			}
			for _, s := range test.writes {
				// io.WriteString takes the WriteString method when there is
				// one, which is the path the text encoder uses.
				if _, err := io.WriteString(w, s); err != nil {
					t.Fatalf("WriteString: %v", err)
				}
			}
			if act := w.endsWithEmptyLine(); act != test.exp {
				t.Errorf("endsWithEmptyLine: %t; want %t", act, test.exp)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("sub", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", "link"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sub/old.bin", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("sub/child", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub/child", "view"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		path string
		exp  string
	}{
		{
			name: "relative new file",
			path: "new.bin",
			exp:  filepath.Join(real, "new.bin"),
		},
		{
			name: "absolute new file",
			path: filepath.Join(dir, "new.bin"),
			exp:  filepath.Join(real, "new.bin"),
		},
		{
			name: "new file through a directory symlink",
			path: "link/new.bin",
			exp:  filepath.Join(real, "sub", "new.bin"),
		},
		{
			name: "existing file through a directory symlink",
			path: "link/old.bin",
			exp:  filepath.Join(real, "sub", "old.bin"),
		},
		{
			name: "new file through dot-dot behind a directory symlink",
			path: "view/../new.bin",
			exp:  filepath.Join(real, "sub", "new.bin"),
		},
		{
			name: "existing file through dot-dot behind a directory symlink",
			path: "view/../old.bin",
			exp:  filepath.Join(real, "sub", "old.bin"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if act := resolve(test.path); act != test.exp {
				t.Errorf("resolve(%q): %q; want %q", test.path, act, test.exp)
			}
		})
	}
}
