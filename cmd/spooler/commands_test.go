package main

import (
	"context"
	"testing"

	"github.com/gobwas/cli"
	"github.com/google/go-cmp/cmp"
)

func testGroup() *commandGroup {
	noop := cli.CommandFunc(func(context.Context, []string) error {
		return nil
	})
	return mustGroup(commandGroup{
		cmds: cli.Commands{
			"send":     noop,
			"recv":     noop,
			"messages": noop,
			"queues":   noop,
		},
		order: []string{"send", "recv", "messages", "queues"},
		aliases: map[string]string{
			"receive": "recv",
		},
	})
}

func TestCommandGroupResolve(t *testing.T) {
	g := testGroup()
	for _, test := range []struct {
		name       string
		args       []string
		expArgs    []string
		expUnknown string
	}{
		{
			name:    "command",
			args:    []string{"send", "-q", "jobs"},
			expArgs: []string{"send", "-q", "jobs"},
		},
		{
			name:    "alias",
			args:    []string{"receive", "-q", "jobs"},
			expArgs: []string{"recv", "-q", "jobs"},
		},
		{
			name:    "alias after help",
			args:    []string{"help", "receive"},
			expArgs: []string{"help", "recv"},
		},
		{
			name:       "unknown",
			args:       []string{"sned"},
			expArgs:    []string{"sned"},
			expUnknown: "sned",
		},
		{
			name:    "help alone",
			args:    []string{"help"},
			expArgs: []string{"help"},
		},
		{
			name:    "nothing",
			args:    nil,
			expArgs: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			args, unknown := g.resolve(test.args)
			if diff := cmp.Diff(test.expArgs, args); diff != "" {
				t.Errorf("args:\n%s", diff)
			}
			if unknown != test.expUnknown {
				t.Errorf("unknown: %q; want %q", unknown, test.expUnknown)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		exp  result
	}{
		{
			name: "under a group",
			args: []string{"queues", "lst"},
			exp: result{
				stderr: "`spooler queues lst`: unknown command\nRun `spooler" +
					" queues help` for usage.\n",
				code: 2,
			},
		},
		{
			name: "at the root",
			args: []string{"login"},
			exp: result{
				stderr: "`spooler login`: unknown command\nRun `spooler help`" +
					" for usage.\n",
				code: 2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			expect(t, run(t, &fakeClient{}, "", test.args...), test.exp)
		})
	}
}

func TestCommandGroupDescription(t *testing.T) {
	named := &cli.Container{
		Command: cli.CommandFunc(func(context.Context, []string) error {
			return nil
		}),
		DoSummary: func() string {
			return "send a message"
		},
	}
	g := mustGroup(commandGroup{
		cmds: cli.Commands{
			"send": named,
			"recv": cli.CommandFunc(func(context.Context, []string) error {
				return nil
			}),
		},
		order: []string{"send", "recv"},
	})
	exp := "Commands:\n\n  send  send a message\n  recv"
	if act := g.Description(); act != exp {
		t.Errorf("description:\n%s\nwant:\n%s", act, exp)
	}
}
