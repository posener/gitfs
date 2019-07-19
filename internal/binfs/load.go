package binfs

import (
	"go/ast"
	"go/token"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// fsProviderFn is a function that given a project name it returns
// its filesystem.
type fsProviderFn func(project string) (http.FileSystem, error)

// LoadBinaries load all binaries in the files according to the defined patterns.
// The returned map maps project name that is used in any of the files that matched
// any of the pattern to its binary encoded content.
func LoadBinaries(patterns []string, provider fsProviderFn) (map[string]string, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, patterns...)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	// Check if any file was loaded.
	totalFiles := 0
	for _, pkg := range pkgs {
		totalFiles += len(pkg.Syntax)
	}
	if totalFiles == 0 {
		return nil, errors.New("no packages were loaded")
	}

	l := loader{
		provider: provider,
		binaries: make(map[string]string),
	}

	// Inspect all loaded files.
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			l.inspectFile(file, pkg.Fset)
		}
	}
	return l.binaries, nil
}

type loader struct {
	// binaries maps a project name to its binary representation.
	binaries map[string]string
	// provider is a function that provides a filesystem given a
	// project name.
	provider fsProviderFn
}

// inspectFile inspects a single file and looks for `gitfs.New` calls.
// If a call was found, it loads the project in the call and save it in its
// binary format in the binaries dict.
func (l *loader) inspectFile(file *ast.File, fset *token.FileSet) {
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			var id *ast.Ident
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				id = fun
			case *ast.SelectorExpr:
				id = fun.Sel
			}
			if id != nil && id.Name == "New" {
				if isPkgDot(call.Fun, "gitfs", "New") {
					project := projectName(call.Args[1])
					if project == "" {
						pos := fset.Position(call.Pos())
						log.Printf(
							"Skipping gitfs.New call in %s. Could not get project name from call.",
							pos)
						return false
					}
					if l.binaries[project] != "" {
						return false
					}
					l.binaries[project] = l.projectBinary(project)
				}
			}
		}
		return true
	})
}

// projectBinary retruns the binary encoded format of a single project.
func (l *loader) projectBinary(project string) string {
	log.Printf("Encoding project: %s", project)
	fs, err := l.provider(project)
	if err != nil {
		log.Printf("Failed creating filesystem %q: %s", project, err)
		return ""
	}
	b, err := encode(fs)
	if err != nil {
		log.Printf("Failed encoding filesystem %q: %s", project, err)
		return ""
	}
	return string(b)
}

// isPkgDot returns true if expr is `<pkg>.<name>`
func isPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && isIdent(sel.Sel, name)
}

// isIdent returns true if expr is `<ident>`.
func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

// projectName takes the Expr that represent the second argument to the gitfs.New call
// and converts it to its content.
func projectName(expr ast.Expr) string {
	switch arg := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(arg.Value, `"`)
	}
	return ""
}
