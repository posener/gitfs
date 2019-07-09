package githubfs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/tree"
	"github.com/posener/gitfs/log"
)

var (
	reGithubProject = regexp.MustCompile(`^github\.com/([^@/]+)/([^@/]+)(/([^@]*))?(@([^#]+))?$`)
	reSemver        = regexp.MustCompile(`^v\d+(\.\d+){0,2}$`)
)

type project struct {
	client *github.Client
	owner  string
	repo   string
	ref    string
	path   string
}

// Match returns true if the given projectName matches a github project.
func Match(projectName string) bool {
	return reGithubProject.MatchString(projectName)
}

// New returns a Tree for a given github project name.
func New(ctx context.Context, client *http.Client, projectName string) (*tree.FS, error) {
	p, err := newGithubProject(ctx, client, projectName)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	t, err := p.getTree(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("Loaded project with %d files in %.1fs", len(t), time.Now().Sub(start).Seconds())
	return tree.NewFS(t), nil
}

func newGithubProject(ctx context.Context, client *http.Client, projectName string) (*project, error) {
	if client == nil {
		client = http.DefaultClient
	}
	p := &project{client: github.NewClient(client)}
	var err error

	p.owner, p.repo, p.path, p.ref, err = githubProjectProperties(projectName)
	if err != nil {
		return nil, err
	}
	// Set ref to default branch in case it is empty.
	if p.ref == "" {
		repo, _, err := p.client.Repositories.Get(ctx, p.owner, p.repo)
		if err != nil {
			return nil, errors.Wrap(err, "get git repository")
		}
		p.ref = "heads/" + repo.GetDefaultBranch()
	}
	return p, nil
}

func githubProjectProperties(projectName string) (owner, repo, path, ref string, err error) {
	matches := reGithubProject.FindStringSubmatch(projectName)
	if len(matches) < 2 {
		err = fmt.Errorf("bad project name: %s", projectName)
		return
	}
	owner = matches[1]
	repo = matches[2]
	path = matches[4]
	ref = matches[6]

	// Add "/" suffix to path.
	if len(path) > 0 && path[len(path)-1] != '/' {
		path = path + "/"
	}

	// If ref is semver, add tags/ prefix to make it a valid ref.
	if reSemver.MatchString(ref) {
		ref = "tags/" + ref
	}

	err = verifyRef(ref)
	return
}

func (p *project) getTree(ctx context.Context) (tree.Tree, error) {
	gitTree, _, err := p.client.Git.GetTree(ctx, p.owner, p.repo, p.ref, true)
	if err != nil {
		return nil, errors.Wrap(err, "get git tree")
	}
	t := make(tree.Tree)
	for _, entry := range gitTree.Entries {
		path := entry.GetPath()
		if p.path != "" {
			if !strings.HasPrefix(path, p.path) {
				continue
			}
			path = strings.TrimPrefix(path, p.path)
		}

		var err error
		switch entry.GetType() {
		case "tree": // A directory.
			err = t.AddDir(path)
		case "blob": // A file.
			err = t.AddFile(path, entry.GetSize(), p.contentLoader(entry.GetSHA()))
		}
		if err != nil {
			return nil, errors.Wrapf(err, "adding %s", path)
		}
	}
	return t, nil
}

// contentLoader gets content of git blob according to git sha of that blob.
func (p *project) contentLoader(sha string) func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		blob, _, err := p.client.Git.GetBlob(ctx, p.owner, p.repo, sha)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting blob")
		}
		switch encoding := blob.GetEncoding(); encoding {
		case "base64":
			return base64.StdEncoding.DecodeString(blob.GetContent())
		default:
			return nil, errors.Errorf("unexpected encoding: %s", encoding)
		}
	}
}

func verifyRef(ref string) error {
	if ref != "" && !strings.HasPrefix(ref, "heads/") && !strings.HasPrefix(ref, "tags/") {
		return errors.New("ref must have a 'heads/' or 'tags/' prefix")
	}
	return nil
}
