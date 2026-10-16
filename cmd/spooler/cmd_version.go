package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"runtime/debug"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

// version is the CLI's release, set by the release build with
// -ldflags "-X main.version=v0.1.0"; empty means a go install or a local
// build, whose version the module info gives.
var version string

type versionCommand struct {
	output output[versionOutput]
}

func (c *versionCommand) Summary() string {
	return "Print the CLI's version and the SDK's"
}

func (c *versionCommand) Synopsis() string {
	return "[<options>]"
}

//go:embed cmd_version.txt
var versionDescText string

var versionDesc *template.Template

func init() {
	versionDesc = must(template.New("").Parse(versionDescText))
}

func (c *versionCommand) Description() string {
	return renderDesc(versionDesc)
}

func (c *versionCommand) DefineFlags(fs *flag.FlagSet) {
	c.output.defineFlags(fs)
}

func (c *versionCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	if err := enc.Encode(versionOutput{
		CLI: cliVersion(),
		SDK: spooler.Version,
	}); err != nil {
		return err
	}
	return enc.Flush()
}

// cliVersion is the release set at build time, else the module version a
// go install recorded, else devel for a build from a checkout.
func cliVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "devel"
}
