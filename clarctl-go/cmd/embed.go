// cmd/embed.go — bundles the runbooks directory into the binary.
package cmd

import "embed"

//go:embed embed_runbooks/*
var embeddedRunbooks embed.FS

func init() {
	RunbookFiles = embeddedRunbooks
}
