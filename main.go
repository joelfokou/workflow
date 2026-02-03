// Package main is the entry point for the WorkFlow CLI application.
// It initialises and executes the command-line interface by calling cmd.Execute().
package main

import (
	"github.com/joelfokou/workflow/cmd"
)

// main is the entry point of the application.
// It delegates execution to the cmd package's Execute function.
func main() {
	cmd.Execute()
}
