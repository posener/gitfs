// gitfs command line tool, for generating binary conetent of the used filesystems.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"

	"github.com/posener/gitfs"
	"github.com/posener/gitfs/fsutil"
	"github.com/posener/gitfs/internal/binfs"
)

var (
	out         = flag.String("out", "gitfs.go", "Output file")
	pkg         = flag.String("pkg", "", "Package name for output file (default is the package name of current directory)")
	skipTestGen = flag.Bool("skip-test-gen", false, "Skip test generation")
	bootstrap   = flag.Bool("bootstrap", false, "Bootstrap mode. For package internal usage.")
)

// templates are used for the generated files. They
// are loaded with loadTemplate function call.
var templates *template.Template

func main() {
	// Parse flags
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("At least one file pattern should be provided.")
	}

	gitfs.SetLogger(log.New(os.Stderr, "[gitfs] ", log.LstdFlags))
	loadTemplates()

	// Fix flags.
	var err error
	*out, err = getOut(*out)
	if err != nil {
		log.Fatalf("Invalid out flag: %s", err)
	}
	*pkg, err = getPkg(*pkg, *out)
	if err != nil {
		log.Fatalf("Invalid: pkg must be provided if output is not a Go package: %s", err)
	}

	calls, err := binfs.LoadCalls(flag.Args()...)
	if err != nil {
		log.Fatalf("Failed loading binaries: %s", err)
	}
	if len(calls) == 0 {
		log.Fatalf("Did not found any calls for gitfs.New")
	}

	binaries := binfs.GenerateBinaries(calls, provider)

	// Generate output
	createOut(binaries)
	createTest(calls)
}

func createOut(binaries map[string]string) {
	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("Failed creating file %q: %s", *out, err)
	}
	defer f.Close()

	err = generate(f, binaries)
	if err != nil {
		defer os.Remove(*out)
		log.Fatalf("Failed generating filesystem: %s", err)
	}
	defer goimports(*out)
}

func createTest(calls binfs.Calls) {
	if *skipTestGen {
		log.Print("Skipping test generation")
		return
	}
	testPath := strings.TrimSuffix(*out, ".go") + "_test.go"
	f, err := os.Create(testPath)
	if err != nil {
		log.Fatalf("Failed creating file %q: %s", testPath, err)
	}
	defer f.Close()

	testName := strings.Title(strings.TrimSuffix(filepath.Base(*out), ".go"))

	err = generateTest(f, calls, testName)
	if err != nil {
		defer os.Remove(testPath)
		log.Fatalf("Failed generating tests: %s", err)
	}
	defer goimports(testPath)
}

func generate(w io.Writer, binaries map[string]string) error {
	return templates.ExecuteTemplate(w, "binary.go.gotmpl", struct {
		Package  string
		Binaries map[string]string
		Version  int
	}{
		Package:  *pkg,
		Binaries: binaries,
		Version:  binfs.EncodeVersion,
	})
}

func generateTest(w io.Writer, calls binfs.Calls, testName string) error {
	return templates.ExecuteTemplate(w, "test.go.gotmpl", struct {
		Package  string
		Calls    binfs.Calls
		TestName string
	}{
		Package:  *pkg,
		Calls:    calls,
		TestName: testName,
	})
}

// getOut fixes out variable if it points to a directory or a file
// non-existing directory.
func getOut(out string) (string, error) {
	if out == "" {
		return "gitfs.go", nil
	}
	st, err := os.Stat(out)
	if err != nil {
		// File does not exists, make sure it is a file in current directory
		// or other existing directory.
		if !strings.HasSuffix(out, ".go") {
			return "", errors.New("output file should be a go file")
		}
		dir, _ := filepath.Split(out)
		if dir == "" {
			// The user chose to create a local file.
			return out, nil
		}
		// The user creates a file in directory `dir`.
		st, err := os.Stat(dir)
		if err != nil {
			return "", errors.Errorf("output directory %q not found: %s", dir, err)
		}
		if !st.IsDir() {
			return "", errors.Errorf("output directory %q is not a directory", dir)
		}
		return out, nil
	}
	if st.IsDir() {
		// If the given output is a directory, add filename 'gitfs.go'.
		out = filepath.Join(out, "gitfs.go")
	}
	return out, nil
}

// getPkg fixes the package name according to the given name in the
// command line or the package of the output file.
func getPkg(pkg, out string) (string, error) {
	if pkg != "" {
		return pkg, nil
	}
	outDir, _ := filepath.Split(out)
	if outDir == "" {
		outDir = "."
	}
	pkgs, err := packages.Load(nil, outDir)
	if err != nil {
		return "", errors.Errorf("failed loading package in %q: %s", outDir, err)
	}
	if len(pkgs) == 0 {
		return "", errors.Errorf("could not load package in %q", outDir)
	}
	return pkgs[0].Name, nil
}

func goimports(path string) {
	err := exec.Command("goimports", "-w", path).Run()
	if err != nil {
		log.Printf("Failed goimports on %s: %s", path, err)
	}
}

//go:generate go run . -bootstrap -out templates.go $GOFILE
func loadTemplates() {
	// For bootstrapping purposes, an environment variable must be set
	// such that the template themselves will be loaded from local path
	// when they are generating their own template.
	local := ""
	if *bootstrap {
		log.Println("Bootstrapping gitfs templates.")
		local = "."
	}
	fs, err := gitfs.New(context.Background(),
		"github.com/posener/gitfs/cmd/gitfs/templates", gitfs.OptLocal(local))
	if err != nil {
		panic(err)
	}
	templates = template.Must(fsutil.TmplParseGlob(fs, nil, "*"))
}

const usage = `gitfs packs filesystems into Go binary for github.com/posener/gitfs library.

Usage:

	gitfs <flags> <patterns>

The command will traverses all Go files in the given patterns and
looks for 'gitfs.New' calls. For each of these calls, it downloads the
specified project. All the projects are then saved into a single
go file (default gitfs.go).
When this file is compiled to a Go binary, the projects are automatically
loaded from the packed version instead of remote repository.

Note:

The calls for 'gitfs.New' must contain an explicit string represented
project name. With the current implementation, the project can't be
inferred from a variable or a constant.


Example:

To pack all usage of gitfs filesystems in the current project, run from
the root of the project the following command:

	gitfs ./...

Flags:

`
