package githubfs

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/glob"
	"github.com/posener/gitfs/internal/log"
	"github.com/posener/gitfs/internal/tree"
)

type githubfs struct {
	*project
	client     *github.Client
	httpClient *http.Client
	glob       glob.Patterns
}

type treeGetter interface {
	get(context.Context) (tree.Tree, error)
}

// Match returns true if the given projectName matches a github project.
func Match(projectName string) bool {
	return reGithubProject.MatchString(projectName)
}

// New returns a Tree for a given github project name.
func New(ctx context.Context, client *http.Client, projectName string, prefetch bool, glob []string) (tree.Tree, error) {
	fs, err := newGithubFS(ctx, client, projectName, glob)
	if err != nil {
		return nil, err
	}
	var t tree.Tree

	// Log tree construction time.
	defer func(start time.Time) {
		log.Printf("Loaded project %q with %d files in %.1fs", projectName, len(t), time.Now().Sub(start).Seconds())
	}(time.Now())

	var getter treeGetter
	if prefetch {
		g := getContents(*fs)
		getter = &g
	} else {
		g := getATree(*fs)
		getter = &g
	}
	return getter.get(ctx)
}

func newGithubFS(ctx context.Context, client *http.Client, projectName string, patterns []string) (*githubfs, error) {
	g, err := glob.New(patterns...)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	project, err := newProject(projectName)
	if err != nil {
		return nil, err
	}

	fs := &githubfs{
		project:    project,
		client:     github.NewClient(client),
		httpClient: client,
		glob:       g,
	}

	// Set ref to default branch in case it is empty.
	if fs.ref == "" {
		repo, _, err := fs.client.Repositories.Get(ctx, fs.owner, fs.repo)
		if err != nil {
			return nil, errors.Wrap(err, "get git repository")
		}
		fs.ref = "heads/" + repo.GetDefaultBranch()
	}
	return fs, nil
}
