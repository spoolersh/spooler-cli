package flagvars

import (
	"flag"
	"io"
	"strconv"
	"testing"
)

func TestIsBoolFlag(t *testing.T) {
	var (
		optBool  *bool
		optInt   *int
		convBool bool
		convStr  string
		given    bool
	)
	for _, test := range []struct {
		name  string
		value flag.Value
		exp   bool
	}{
		{
			name:  "opt var of bool",
			value: OptVar(&optBool, strconv.ParseBool),
			exp:   true,
		},
		{
			name:  "opt var of int",
			value: OptVar(&optInt, strconv.Atoi),
		},
		{
			name:  "conv func var of bool",
			value: ConvFuncVar(&convBool, strconv.ParseBool),
			exp:   true,
		},
		{
			name:  "conv func var of string",
			value: ConvFuncVar(&convStr, Ident[string]),
		},
		{
			name:  "opt around a bool",
			value: Opt(OptVar(&optBool, strconv.ParseBool), &given),
			exp:   true,
		},
		{
			name:  "validate around a string",
			value: Validate(ConvFuncVar(&convStr, Ident[string])),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if act := isBoolFlag(test.value); act != test.exp {
				t.Errorf("IsBoolFlag: %t; want %t", act, test.exp)
			}
		})
	}
	t.Run("parsed without a value", func(t *testing.T) {
		var p *bool
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		fs.Var(OptVar(&p, strconv.ParseBool), "flag", "")
		if err := fs.Parse([]string{"-flag"}); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if p == nil || !*p {
			t.Errorf("value: %v; want true", p)
		}
	})
}

func TestOpt(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		value string
		given bool
		err   bool
	}{
		{
			name:  "not given",
			args:  nil,
			value: "def",
			given: false,
		},
		{
			name:  "given",
			args:  []string{"-name", "val"},
			value: "val",
			given: true,
		},
		{
			name: "given wrong",
			args: []string{"-name", "bad"},
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				value = "def"
				given bool
			)
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			fs.Var(Opt(Validate(
				ConvFuncVar(&value, Ident[string]),
				func(s string) error {
					if s == "bad" {
						return io.ErrUnexpectedEOF
					}
					return nil
				},
			), &given), "name", "")
			if act, exp := fs.Lookup("name").DefValue, "def"; act != exp {
				t.Errorf("default: %q; want %q", act, exp)
			}
			err := fs.Parse(test.args)
			if (err != nil) != test.err {
				t.Fatalf("Parse(%q): %v; want error: %t", test.args, err, test.err)
			}
			if test.err {
				return
			}
			if value != test.value {
				t.Errorf("value: %q; want %q", value, test.value)
			}
			if given != test.given {
				t.Errorf("given: %t; want %t", given, test.given)
			}
		})
	}
}
