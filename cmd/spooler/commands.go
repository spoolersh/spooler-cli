package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gobwas/cli"
)

// commandGroup is a cli.Commands with the help list in the order of the
// work rather than alphabetical, aliases that help does not list, and a
// two-line refusal of an unknown command.
type commandGroup struct {
	// summary is the group's one line in its parent's list.
	summary string

	// prolog opens the group's help.
	prolog string

	cmds cli.Commands

	// order names every command of cmds, in the order help lists them.
	order []string

	// aliases maps an alias to the command it names.
	aliases map[string]string

	// epilog is text help prints after the flags, where GNU tools put
	// their exit status.
	epilog string
}

var _ interface {
	cli.Command
	cli.SummaryProvider
	cli.PrologProvider
	cli.SynopsisProvider
	cli.DescriptionProvider
	cli.EpilogProvider
} = (*commandGroup)(nil)

// mustGroup checks that order and cmds agree, so they cannot drift apart,
// and that every alias names a command.
func mustGroup(g commandGroup) *commandGroup {
	for name := range g.cmds {
		if !slices.Contains(g.order, name) {
			panic(fmt.Sprintf(
				"command %q is not in the help order",
				name,
			))
		}
	}
	for _, name := range g.order {
		if _, has := g.cmds[name]; !has {
			panic(fmt.Sprintf(
				"command %q in the help order does not exist",
				name,
			))
		}
	}
	for alias, name := range g.aliases {
		if _, has := g.cmds[name]; !has {
			panic(fmt.Sprintf(
				"alias %q names no command: %q",
				alias, name,
			))
		}
	}
	return &g
}

// Run resolves an alias, refuses an unknown command in two lines, as the go
// command does, where the usage would be a page for a typo and the help
// it points at lists every command on one screen, and dispatches as
// cli.Commands does.
func (g *commandGroup) Run(ctx context.Context, args []string) error {
	args, unknown := g.resolve(args)
	if unknown != "" {
		return cli.Exitf(2,
			"`%[1]s %[2]s`: unknown command\nRun `%[1]s help` for usage.",
			commandPath(ctx), unknown,
		)
	}
	return g.cmds.Run(ctx, args)
}

// resolve replaces an alias in the command position of args, which follows
// any leading help words, and returns the name there when it is no
// command of the group.
func (g *commandGroup) resolve(args []string) (ret []string, unknown string) {
	i := 0
	for i < len(args) && args[i] == "help" {
		i++
	}
	if i == len(args) {
		return args, ""
	}
	name := args[i]
	if target, ok := g.aliases[name]; ok {
		ret = slices.Clone(args)
		ret[i] = target
		return ret, ""
	}
	if _, ok := g.cmds[name]; ok {
		return args, ""
	}
	return args, name
}

// Summary implements cli.SummaryProvider.
func (g *commandGroup) Summary() string {
	return g.summary
}

// Prolog implements cli.PrologProvider: the logo, when there is one, then an
// empty line and the prolog.
func (g *commandGroup) Prolog() string {
	return g.prolog
}

// Synopsis implements cli.SynopsisProvider.
func (g *commandGroup) Synopsis() string {
	return g.cmds.Synopsis()
}

// Epilog implements cli.EpilogProvider.
func (g *commandGroup) Epilog() string {
	return g.epilog
}

// Description implements cli.DescriptionProvider: the commands in order,
// each with its summary, in the layout cli uses.
func (g *commandGroup) Description() string {
	width := 0
	for _, name := range g.order {
		width = max(width, utf8.RuneCountInString(name))
	}
	var sb strings.Builder
	sb.WriteString("Commands:\n")
	for _, name := range g.order {
		fmt.Fprintf(&sb, "\n  %s", name)
		// Pad only when there is a summary to align, to leave no trailing
		// whitespace.
		if p, ok := g.cmds[name].(cli.SummaryProvider); ok && p.Summary() != "" {
			pad := width - utf8.RuneCountInString(name) + 2
			fmt.Fprintf(&sb, "%*s%s", pad, "", p.Summary())
		}
	}
	return sb.String()
}

// ensureNoArgs refuses what the parser left over. Every input is a flag, and
// the parser stops at the first word that is not one, so the flags behind it
// were never read.
func ensureNoArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	return cli.Exitf(2,
		"unexpected argument: %q",
		args[0],
	)
}

// noFlags reports whether the running command was given no flag at all,
// from the command line, the profile or the environment: a bare
// invocation, answered with the usage rather than with the first missing
// flag. A flag set from any source counts, since a command whose every
// flag the profile and the environment supply is a complete invocation
// with nothing on its command line.
func noFlags(ctx context.Context) bool {
	infos := cli.ContextCommandsInfo(ctx)
	if len(infos) == 0 {
		return false
	}
	return infos[len(infos)-1].FlagSet.NFlag() == 0
}

// commandPath is the invocation so far, "spooler queues", for a message.
func commandPath(ctx context.Context) string {
	var names []string
	for _, info := range cli.ContextCommandsInfo(ctx) {
		names = append(names, info.Name)
	}
	return strings.Join(names, " ")
}
