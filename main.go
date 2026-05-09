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
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		runUpdate(os.Args[2:])
	case "unused":
		runUnused(os.Args[2:])
	case "license":
		runLicense(os.Args[2:])
	case "clean":
		runClean(os.Args[2:])
	case "css":
		runCSS(os.Args[2:])
	case "-v", "--version", "version":
		fmt.Println("bumpit", buildVersion())
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "bumpit: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	showIndirect := fs.Bool("show-indirect", false, "include indirect Go module dependencies")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Show outdated packages and update them interactively")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit update [flags] [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "      --show-indirect   include indirect Go module dependencies")
		fmt.Fprintln(os.Stderr, "  -h, --help            help for update")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{ShowIndirect: *showIndirect}))
}

func runUnused(args []string) {
	fs := flag.NewFlagSet("unused", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Scan for unused direct dependencies and remove them interactively")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit unused [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for unused")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{UnusedMode: true}))
}

func runLicense(args []string) {
	fs := flag.NewFlagSet("license", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Audit dependency licenses and flag copyleft or unknown licenses")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit license [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for license")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{LicenseMode: true}))
}

func runCSS(args []string) {
	fs := flag.NewFlagSet("css", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Scan CSS/SCSS files for class selectors that are never referenced in templates")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit css [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for css")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{CSSMode: true}))
}

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Find and delete generated artifact directories (node_modules, dist, .next, ...)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bumpit clean [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for clean")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{CleanMode: true}))
}

func rootDir(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	return root
}

func runTUI(model tea.Model) {
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running bumpit: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "bumpit — monorepo chore helper")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  bumpit <command> [flags] [directory]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  update     Show outdated packages and update them interactively")
	fmt.Fprintln(os.Stderr, "  unused     Scan for unused direct dependencies and remove them")
	fmt.Fprintln(os.Stderr, "  license    Audit dependency licenses and flag copyleft or unknown")
	fmt.Fprintln(os.Stderr, "  clean      Find and delete generated artifact directories interactively")
	fmt.Fprintln(os.Stderr, "  css        Find CSS class selectors never referenced in templates")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  -h, --help      help for bumpit")
	fmt.Fprintln(os.Stderr, "  -v, --version   print version and exit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Use \"bumpit <command> --help\" for more information about a command.")
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
