package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcelmettler/bumpit/internal/ui"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	showIndirect := flag.Bool("show-indirect", false, "include indirect Go module dependencies")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "bumpit — interactive dependency updater")
	fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit [flags] [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Key bindings (list view):")
		fmt.Fprintln(os.Stderr, "  j/k     navigate    space  toggle selection")
		fmt.Fprintln(os.Stderr, "  enter   changelog   u      update selected")
		fmt.Fprintln(os.Stderr, "  /       filter      s      cycle sort")
		fmt.Fprintln(os.Stderr, "  a       select all  ?      help overlay")
		fmt.Fprintln(os.Stderr, "  q       quit")
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("bumpit", buildVersion())
		return
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	model := ui.New(root, ui.Config{ShowIndirect: *showIndirect})

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running bumpit: %v\n", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
