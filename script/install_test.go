package script

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// repo is the release URL's root, as install.sh builds it.
const repo = "https://github.com/spoolersh/spooler-cli"

// stubCurl stands in for curl on PATH: it serves the fixture release under
// the latest URL and under its tag, logs every URL, and knows no other
// release, failing as curl -f does on a 404.
const stubCurl = `#!/bin/sh
out=
while [ $# -gt 1 ]; do
	if [ "$1" = -o ]; then
		out=$2
		shift
	fi
	shift
done
echo "$1" >>"$CURL_LOG"
for base in $FIXTURE_URLS; do
	if [ "${1%/*}" = "$base" ] && [ -f "$FIXTURES/${1##*/}" ]; then
		cp "$FIXTURES/${1##*/}" "$out"
		exit 0
	fi
done
exit 22
`

// result is what one run of install.sh produced: the exit code, both
// streams, the URLs fetched in order, and the destination's state: the
// installed binary's output, "directory" for one, "" for nothing there.
type result struct {
	code      int
	stdout    string
	stderr    string
	urls      string
	installed string
}

// TestInstall runs install.sh against a fixture release served by a stub
// curl. Expectations name the install directory DIR, the archive NAME, its
// checksum SUM and a corrupted archive's checksum BAD.
func TestInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is for Linux and macOS")
	}
	name := "spooler-cli_0.1.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	for _, test := range []struct {
		name string
		// version is SPOOLER_VERSION; empty is unset.
		version string
		// corrupt appends a byte to the archive after the checksums are
		// written.
		corrupt bool
		// noArchive lists an archive for another platform only.
		noArchive bool
		// dirAtDest puts a directory where the binary would go.
		dirAtDest bool
		exp       result
	}{
		{
			// The latest release is resolved once, through checksums.txt,
			// and the archive is fetched from that release by its tag.
			name: "latest",
			exp: result{
				stdout: "installed spooler v0.1.0 to DIR/spooler\n" +
					"DIR is not on PATH\n",
				urls: repo + "/releases/latest/download/checksums.txt\n" +
					repo + "/releases/download/v0.1.0/NAME\n",
				installed: "stand-in\n",
			},
		},
		{
			name:    "pinned version",
			version: "0.1.0",
			exp: result{
				stdout: "installed spooler v0.1.0 to DIR/spooler\n" +
					"DIR is not on PATH\n",
				urls: repo + "/releases/download/v0.1.0/checksums.txt\n" +
					repo + "/releases/download/v0.1.0/NAME\n",
				installed: "stand-in\n",
			},
		},
		{
			name:    "pinned version with a v",
			version: "v0.1.0",
			exp: result{
				stdout: "installed spooler v0.1.0 to DIR/spooler\n" +
					"DIR is not on PATH\n",
				urls: repo + "/releases/download/v0.1.0/checksums.txt\n" +
					repo + "/releases/download/v0.1.0/NAME\n",
				installed: "stand-in\n",
			},
		},
		{
			name:    "missing release",
			version: "9.9.9",
			exp: result{
				code: 1,
				stderr: "install.sh: no release found at " +
					repo + "/releases/download/v9.9.9\n",
				urls: repo + "/releases/download/v9.9.9/checksums.txt\n",
			},
		},
		{
			name:    "corrupt archive",
			corrupt: true,
			exp: result{
				code:   1,
				stderr: "install.sh: checksum of NAME is BAD; want SUM\n",
				urls: repo + "/releases/latest/download/checksums.txt\n" +
					repo + "/releases/download/v0.1.0/NAME\n",
			},
		},
		{
			name:      "no archive for the platform",
			noArchive: true,
			exp: result{
				code: 1,
				stderr: "install.sh: no archive for " +
					runtime.GOOS + "/" + runtime.GOARCH + " in the release\n",
				urls: repo + "/releases/latest/download/checksums.txt\n",
			},
		},
		{
			// Refused before anything is fetched, and left as it was.
			name:      "directory at the destination",
			dirAtDest: true,
			exp: result{
				code:      1,
				stderr:    "install.sh: DIR/spooler is a directory\n",
				installed: "directory",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				tmp = t.TempDir()
				fxs = filepath.Join(tmp, "release")
				bin = filepath.Join(tmp, "bin")
				dir = filepath.Join(tmp, "install")
				log = filepath.Join(tmp, "curl.log")
			)
			for _, d := range []string{fxs, bin} {
				if err := os.Mkdir(d, 0755); err != nil {
					t.Fatal(err)
				}
			}
			archive, err := targz("spooler",
				`#!/bin/sh
				echo stand-in
				`,
			)
			if err != nil {
				t.Fatal(err)
			}
			var (
				sum    = sha256hex(archive)
				listed = name
			)
			if test.noArchive {
				listed = "spooler-cli_0.1.0_plan9_mips.tar.gz"
			}
			err = os.WriteFile(filepath.Join(fxs, "checksums.txt"),
				[]byte(sum+"  "+listed+"\n"), 0644,
			)
			if err != nil {
				t.Fatal(err)
			}

			var bad string
			if test.corrupt {
				archive = append(archive, 'x')
				bad = sha256hex(archive)
			}
			err = os.WriteFile(filepath.Join(fxs, name), archive, 0644)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(filepath.Join(bin, "curl"), []byte(stubCurl), 0755)
			if err != nil {
				t.Fatal(err)
			}
			if test.dirAtDest {
				err := os.MkdirAll(filepath.Join(dir, "spooler"), 0755)
				if err != nil {
					t.Fatal(err)
				}
			}

			cmd := exec.Command("sh", "install.sh")
			cmd.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"SPOOLER_INSTALL_DIR="+dir,
				"CURL_LOG="+log,
				"FIXTURES="+fxs,
				"FIXTURE_URLS="+
					repo+"/releases/latest/download "+
					repo+"/releases/download/v0.1.0",
				// Set for every case, empty included, so the caller's shell
				// cannot pin a version on a case that tests the latest.
				"SPOOLER_VERSION="+test.version,
			)
			var (
				stdout bytes.Buffer
				stderr bytes.Buffer
			)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()
			var code int
			if ee, ok := errors.AsType[*exec.ExitError](err); ok {
				code = ee.ExitCode()
				err = nil
			}
			if err != nil {
				t.Fatalf("run install.sh: %v", err)
			}
			act := result{
				stdout: stdout.String(),
				stderr: stderr.String(),
				code:   code,
			}
			if urls, err := os.ReadFile(log); err == nil {
				act.urls = string(urls)
			}
			// The destination's state: absent, a directory, or a binary that
			// runs; one that exists and does not run is a failure, not
			// an absence.
			target := filepath.Join(dir, "spooler")
			switch info, err := os.Stat(target); {
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				t.Fatal(err)
			case info.IsDir():
				act.installed = "directory"
			default:
				out, err := exec.Command(target).Output()
				if err != nil {
					t.Fatalf("run %s: %v", target, err)
				}
				act.installed = string(out)
			}

			exp := test.exp
			r := strings.NewReplacer(
				"DIR", dir,
				"NAME", name,
				"SUM", sum,
				"BAD", bad,
			)
			exp.stdout = r.Replace(exp.stdout)
			exp.stderr = r.Replace(exp.stderr)
			exp.urls = r.Replace(exp.urls)

			opts := cmp.AllowUnexported(result{})
			if !cmp.Equal(exp, act, opts) {
				t.Error(cmp.Diff(exp, act, opts))
			}
		})
	}
}

func targz(name, content string) ([]byte, error) {
	var (
		buf = bytes.NewBuffer(nil)
		gz  = gzip.NewWriter(buf)
		tw  = tar.NewWriter(gz)
	)
	err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(content)),
	})
	if err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
