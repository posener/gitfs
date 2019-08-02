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

// Calls is a map of project to load configuration.
type Calls map[string]*Config

// Config is a configuration for generating a filesystem.
type Config struct {
	Project string
	// globPatterns is a union of all globPatterns that found in all calls
	// for this project.
	globPatterns []string
	// a helper field, used to indicate if there was a project import without
	// a usage of pattern (this means that we should not have patterns applied
	// in the binary creation).
	noPatterns bool
}

// GlobPatterns that should be used for this project.
func (c *Config) GlobPatterns() []string {
	if c.noPatterns {
		return nil
	}
	return c.globPatterns
}

// fsProviderFn is a function that given a project name it returns
// its filesystem.
type fsProviderFn func(c Config) (http.FileSystem, error)

// LoadCalls load all calls to gitfs.New in the files according to the defined patterns.
func LoadCalls(patterns ...string) (Calls, error) {
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
	c := make(Calls)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			c.lookupAST(file, pkg.Fset)
		}
	}
	return c, nil
}

// GenerateBinaries generate binary representation to all given calls.
// The returned map maps project name that is used in any of the files that matched
// any of the pattern to its binary encoded content.
func GenerateBinaries(c Calls, provider fsProviderFn) map[string]string {
	// Load all binaries
	binaries := make(map[string]string)
	for project, config := range c {
		binaries[project] = loadBinary(provider, *config)
	}
	return binaries
}

// lookupAST inspects a single AST and looks for `gitfs.New` calls.
// If a call was found, it saves the project this call was called for
// and options it was called with.
func (c Calls) lookupAST(file *ast.File, fset *token.FileSet) {
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
					pos := fset.Position(call.Pos())
					if project == "" {
						log.Printf(
							"Skipping gitfs.New call in %s. Could not get project name from call.",
							pos)
						return false
					}

					// Mark that project is used.
					if c[project] == nil {
						c[project] = &Config{Project: project}
					}

					// Treat OptGlob call.
					patterns, err := findOptGlob(call.Args[2:])
					if err != nil {
						log.Printf(
							"Failed getting glob options in %s, building without glob pattern: %s",
							pos, err)
						patterns = nil
					}
					if len(patterns) == 0 {
						// This call does not use pattern. Mark it so we will later load
						// all files.
						c[project].noPatterns = true
					} else {
						// Accumulate all the patterns that are used for all the places
						// that the project was used.
						c[project].globPatterns = append(c[project].globPatterns, patterns...)
					}
				}
			}
		}
		return true
	})
}

// projectBinary retruns the binary encoded format of a single project.
func loadBinary(provider fsProviderFn, c Config) string {
	log.Printf("Encoding project: %s", c.Project)
	fs, err := provider(c)
	if err != nil {
		log.Printf("Failed creating filesystem %q: %s", c.Project, err)
		return ""
	}
	b, err := encode(fs)
	if err != nil {
		log.Printf("Failed encoding filesystem %q: %s", c.Project, err)
		return ""
	}
	return string(b)
}

// findOptGlob takes arguments of the gitfs.New and looks for the
// gitfs.OptGlob option. If it finds it, it returns the arguments that
// were passed to that option.
func findOptGlob(exprs []ast.Expr) ([]string, error) {
	for _, expr := range exprs {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}
		if !isPkgDot(call.Fun, "gitfs", "OptGlob") {
			continue
		}
		var patterns []string
		for i, arg := range call.Args {
			pattern := stringExpr(arg)
			if pattern == "" {
				return nil, errors.Errorf(
					"can't understand string expression of OptGlob arg #%d with value %+v",
					i, arg)
			}
			patterns = append(patterns, pattern)
		}
		return patterns, nil
	}
	return nil, nil
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
