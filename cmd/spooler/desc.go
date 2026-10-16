package main

import (
	"fmt"
	"strings"
	"text/template"
)

// A command's description for --help is a text/template in its cmd_*.txt
// file, embedded and parsed at init; the template renders with descFuncs,
// whose methods are the functions it may call.

// descFuncs is what a description renders with.
type descFuncs struct{}

// OpDocsURL is the docs page of an operation, by its operationId.
func (descFuncs) OpDocsURL(op string) string {
	return "https://docs.spooler.sh/operations/" + op
}

// renderDesc renders a description for --help, without the newline the
// file ends with, which help adds itself.
func renderDesc(t *template.Template) string {
	var sb strings.Builder
	if err := t.Execute(&sb, descFuncs{}); err != nil {
		panic(fmt.Sprintf(
			"render description: %v",
			err,
		))
	}
	return strings.TrimRight(sb.String(), "\n")
}
