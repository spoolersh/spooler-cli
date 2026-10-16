package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse"
	"go.yaml.in/yaml/v3"
)

// profileParser is the flag parser that reads the selected profile into
// dest and sets the flags it holds. It runs between the command line and
// the environment, and a flag the command line set is left as it is, so a
// flag beats the profile and the profile beats a variable.
type profileParser struct {
	dest *profileEntry
}

// Parse loads the profile when fs defines its selection, which only the
// root's flag set does, then gives fs those of the profile's values it
// defines: the connection at the root, the spool on a command.
func (p *profileParser) Parse(_ context.Context, fs parse.FlagSet) error {
	var (
		config  = fs.Lookup("config")
		profile = fs.Lookup("profile")
	)
	if config != nil && profile != nil {
		// We're at the root level, so load the profile configuration.
		err := p.load(config.Value.String(), profile.Value.String())
		if err != nil {
			return err
		}
	}
	// Now set up any flags present at the current command level (either root
	// or subcommand).
	for _, v := range p.dest.values() {
		if fs.Lookup(v.name) == nil {
			continue
		}
		if err := fs.Set(v.name, v.value); err != nil {
			return err
		}
	}
	return nil
}

// load reads the profile name from the file at path into dest.
func (p *profileParser) load(path, name string) error {
	if name == "" {
		return nil
	}
	if path == "" {
		return &flagutil.ParseError{
			Name: "config",
			Err: fmt.Errorf(
				"select profile %q: path is not set",
				name,
			),
		}
	}
	bts, err := os.ReadFile(path)
	if err != nil {
		return &flagutil.ParseError{
			Name: "config",
			Err: fmt.Errorf(
				"select profile %q: %v",
				name, err,
			),
		}
	}
	var ps profiles
	if err := yaml.Unmarshal(bts, &ps); err != nil {
		return &flagutil.ParseError{
			Name: "config",
			Err: fmt.Errorf(
				"select profile %q: parse %s: %v",
				name, path, err,
			),
		}
	}
	entry, has := ps.Profiles[name]
	if !has {
		return &flagutil.ParseError{
			Name: "profile",
			Err: fmt.Errorf(
				"select profile %q: no profile defined in %s",
				name, path,
			),
		}
	}

	*p.dest = entry
	return nil
}

// flagValue is a profile value under the name of the flag it sets.
type flagValue struct {
	name  string
	value string
}

// values lists the flags the entry sets, in the order they are set.
func (e *profileEntry) values() []flagValue {
	var vs []flagValue
	if e.Host != nil {
		vs = append(vs, flagValue{
			name:  "host",
			value: *e.Host,
		})
	}
	if e.APIKey != nil {
		vs = append(vs, flagValue{
			name:  "api-key",
			value: *e.APIKey,
		})
	}
	if e.NoTLS != nil {
		vs = append(vs, flagValue{
			name:  "no-tls",
			value: strconv.FormatBool(*e.NoTLS),
		})
	}
	if e.Spool != nil {
		vs = append(vs, flagValue{
			name:  "spool",
			value: *e.Spool,
		})
	}
	return vs
}

type profiles struct {
	Profiles map[string]profileEntry `yaml:"profiles"`
}

// profileEntry contains profile stored config.
type profileEntry struct {
	Host          *string  `yaml:"host"`
	APIKey        *string  `yaml:"api-key"`
	APIKeyCommand *command `yaml:"api-key-command"`
	NoTLS         *bool    `yaml:"no-tls"`
	Spool         *string  `yaml:"spool"`
}

type command struct {
	Bin  string   `yaml:"bin"`
	Args []string `yaml:"args"`
}
