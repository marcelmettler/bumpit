package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcelmettler/chorekit/internal/ui"
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
	case "todo":
		runTodo(os.Args[2:])
	case "i18n":
		runI18n(os.Args[2:])
	case "env":
		runEnv(os.Args[2:])
	case "audit":
		runAudit(os.Args[2:])
	case "pin":
		runPin(os.Args[2:])
	case "sort":
		runSort(os.Args[2:])
	case "-v", "--version", "version":
		fmt.Println("chorekit", buildVersion())
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "chorekit: unknown command %q\n\n", os.Args[1])
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
		fmt.Fprintln(os.Stderr, "  chorekit update [flags] [directory]")
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
		fmt.Fprintln(os.Stderr, "  chorekit unused [directory]")
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
		fmt.Fprintln(os.Stderr, "  chorekit license [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for license")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{LicenseMode: true}))
}

func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Run a security audit and display vulnerabilities with fix availability")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit audit [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for audit")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{AuditMode: true}))
}

func runPin(args []string) {
	fs := flag.NewFlagSet("pin", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Find dependencies using ^ or ~ version ranges and pin them to exact versions")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit pin [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for pin")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{PinMode: true}))
}

func runEnv(args []string) {
	fs := flag.NewFlagSet("env", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Audit environment variables: find unused .env.example vars and undefined source references")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit env [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Expects .env.example, .env.sample, .env.template, .env.defaults, or .env.schema files.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for env")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{EnvMode: true}))
}

func runI18n(args []string) {
	fs := flag.NewFlagSet("i18n", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Audit translation keys: find unused locale keys and undefined source references")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit i18n [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Expects JSON locale files inside a directory named locales/, i18n/, translations/, or lang/.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for i18n")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{I18nMode: true}))
}

func runTodo(args []string) {
	fs := flag.NewFlagSet("todo", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Scan source files for TODO, FIXME, HACK, and XXX comments")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit todo [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for todo")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{TodoMode: true}))
}

func runCSS(args []string) {
	fs := flag.NewFlagSet("css", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Scan CSS/SCSS files for class selectors that are never referenced in templates")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit css [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for css")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{CSSMode: true}))
}

func runSort(args []string) {
	fs := flag.NewFlagSet("sort", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Sort dependency sections in package.json files alphabetically")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit sort [directory]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Sorts dependencies, devDependencies, peerDependencies, and optionalDependencies.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -h, --help   help for sort")
	}
	_ = fs.Parse(args)
	runTUI(ui.New(rootDir(fs.Args()), ui.Config{SortDepsMode: true}))
}

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Find and delete generated artifact directories (node_modules, dist, .next, ...)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  chorekit clean [directory]")
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
		fmt.Fprintf(os.Stderr, "Error running chorekit: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "chorekit — a tidy toolkit for keeping your codebase in shape")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  chorekit <command> [flags] [directory]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  update     Show outdated packages and update them interactively")
	fmt.Fprintln(os.Stderr, "  unused     Scan for unused direct dependencies and remove them")
	fmt.Fprintln(os.Stderr, "  license    Audit dependency licenses and flag copyleft or unknown")
	fmt.Fprintln(os.Stderr, "  clean      Find and delete generated artifact directories interactively")
	fmt.Fprintln(os.Stderr, "  css        Find CSS class selectors never referenced in templates")
	fmt.Fprintln(os.Stderr, "  todo       Scan for TODO, FIXME, HACK, and XXX comments")
	fmt.Fprintln(os.Stderr, "  i18n       Audit translation keys: unused locale keys and undefined references")
	fmt.Fprintln(os.Stderr, "  env        Audit .env.example vars: unused vars and undefined source references")
	fmt.Fprintln(os.Stderr, "  audit      Run security audit and show vulnerabilities with fix availability")
	fmt.Fprintln(os.Stderr, "  pin        Find ^ and ~ version ranges and pin them to exact versions")
	fmt.Fprintln(os.Stderr, "  sort       Sort dependency sections in package.json files alphabetically")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  -h, --help      help for chorekit")
	fmt.Fprintln(os.Stderr, "  -v, --version   print version and exit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Use \"chorekit <command> --help\" for more information about a command.")
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
