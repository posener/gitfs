package githubfs

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	reGithubProject = regexp.MustCompile(`^github\.com/([^@/]+)/([^@/]+)(/([^@]*))?(@([^#]+))?$`)
	reSemver        = regexp.MustCompile(`^v?\d+(\.\d+){0,2}$`)
)

type project struct {
	owner string
	repo  string
	ref   string
	path  string
}

// newProject parses project name into the different components
// it is composed of.
func newProject(projectName string) (p *project, err error) {
	matches := reGithubProject.FindStringSubmatch(projectName)
	if len(matches) < 2 {
		err = fmt.Errorf("bad project name: %s", projectName)
		return
	}

	p = &project{
		owner: matches[1],
		repo:  matches[2],
		path:  matches[4],
		ref:   matches[6],
	}

	// Add "/" suffix to path.
	if len(p.path) > 0 && p.path[len(p.path)-1] != '/' {
		p.path = p.path + "/"
	}

	// If ref is Semver, add 'tags/' prefix to make it a valid ref.
	if reSemver.MatchString(p.ref) {
		p.ref = "tags/" + p.ref
	}

	err = verifyRef(p.ref)
	return
}

func verifyRef(ref string) error {
	if ref != "" && !strings.HasPrefix(ref, "heads/") && !strings.HasPrefix(ref, "tags/") {
		return errors.New("ref must have a 'heads/' or 'tags/' prefix")
	}
	return nil
}
