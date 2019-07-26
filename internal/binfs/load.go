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
type fsProviderFn func(project string, globPatterns []string) (http.FileSystem, error)

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

	// Find all projects
	l := make(loader)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			l.inspectFile(file, pkg.Fset)
		}
	}

	// Load all binaries
	binaries := make(map[string]string)
	for project, config := range l {
		binaries[project] = provider.loadBinary(project, *config)
	}
	return binaries, nil
}

type loader map[string]*config

// config is a project configuration.
type config struct {
	patterns []string
	noPattern bool
}

// inspectFile inspects a single file and looks for `gitfs.New` calls.
// If a call was found, it finds all project that are used in the file.
func (l loader) inspectFile(file *ast.File, fset *token.FileSet) {
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
					project := stringExpr(call.Args[1])
					if project == "" {
						pos := fset.Position(call.Pos())
						log.Printf(
							"Skipping gitfs.New call in %s. Could not get project name from call.",
							pos)
						return false
					}

					// Mark that project is used.
					if l[project] == nil {
						l[project] = &config{}
					}

					// Treat OptGlob call.
					patterns := findOptGlob(call.Args[2:])
					if len(patterns) == 0 {
						// This call does not use pattern. Mark it so we will later load
						// all files.
						l[project].noPattern = true
					} else {
						// Accumulate all the patterns that are used for all the places
						// that the project was used.
						l[project].patterns = append(l[project].patterns, patterns...)
					}
				}
			}
		}
		return true
	})
}

// projectBinary retruns the binary encoded format of a single project.
func (provider fsProviderFn) loadBinary(project string, c config) string {
	log.Printf("Encoding project: %s", project)
	// If there was one place that did not use a pattern, we should ignore
	// the patterns that were used in other places.
	if c.noPattern {
		c.patterns = nil
	}
	fs, err := provider(project, c.patterns)
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

// findOptGlob takes arguments of the gitfs.New and looks for the
// gitfs.OptGlob option. If it finds it, it returns the arguments that
// were passed to that option.
func findOptGlob(exprs []ast.Expr) []string {
	for _, expr := range exprs {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}
		if !isPkgDot(call.Fun, "gitfs", "OptGlob") {
			continue
		}
		var patterns []string
		for _, arg := range call.Args {
			pattern := stringExpr(arg)
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
		return patterns
	}
	return nil
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

// stringExpr takes the Expr that represent a string and converts it to its content.
func stringExpr(expr ast.Expr) string {
	arg, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	return strings.Trim(arg.Value, `"`)
}
