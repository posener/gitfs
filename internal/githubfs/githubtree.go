package githubfs

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/tree"
	"github.com/posener/gitfs/internal/log"
)

var (
	reGithubProject = regexp.MustCompile(`^github\.com/([^@/]+)/([^@/]+)(/([^@]*))?(@([^#]+))?$`)
	reSemver        = regexp.MustCompile(`^v?\d+(\.\d+){0,2}$`)
)

type project struct {
	client     *github.Client
	httpClient *http.Client
	owner      string
	repo       string
	ref        string
	path       string
}

// Match returns true if the given projectName matches a github project.
func Match(projectName string) bool {
	return reGithubProject.MatchString(projectName)
}

// New returns a Tree for a given github project name.
func New(ctx context.Context, client *http.Client, projectName string, prefetch bool) (tree.Tree, error) {
	p, err := newGithubProject(ctx, client, projectName)
	if err != nil {
		return nil, err
	}
	var t tree.Tree

	// Log tree construction time.
	defer func(start time.Time) {
		log.Printf("Loaded project %q with %d files in %.1fs", projectName, len(t), time.Now().Sub(start).Seconds())
	}(time.Now())

	if prefetch {
		t, err = p.prefetchTree(ctx)
	} else {
		t, err = p.getTree(ctx)
	}
	return t, err
}

func newGithubProject(ctx context.Context, client *http.Client, projectName string) (*project, error) {
	if client == nil {
		client = http.DefaultClient
	}
	p := &project{
		client:     github.NewClient(client),
		httpClient: client,
	}
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

// githubProjectProperties parses project name into the different components
// it is composed of.
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

	// If ref is Semver, add 'tags/' prefix to make it a valid ref.
	if reSemver.MatchString(ref) {
		ref = "tags/" + ref
	}

	err = verifyRef(ref)
	return
}

// getTree gets a structure of a sub-tree of a github repository using the Github
// get-a-tree API: https://developer.github.com/v3/git/trees/#get-a-tree.
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

// prefetchTree download a tree from Github, with all of its files content.
func (p *project) prefetchTree(ctx context.Context) (tree.Tree, error) {
	downloader := recursiveGetContents{
		tree:    make(tree.Tree),
		project: p,
		errors:  make(chan error),
	}

	err := downloader.download(ctx)
	if err != nil {
		return nil, err
	}
	return downloader.tree, nil
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

// contentDownloadLoader is a Loader for downling a file from a URL.
// It immediately loads the file rather than lazily.
func (p *project) contentDownloadLoader(ctx context.Context, downloadURL string) func(ctx context.Context) ([]byte, error) {
	var data []byte
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err == nil {
		resp, err := p.httpClient.Do(req.WithContext(ctx))
		if err == nil {
			if resp.StatusCode != http.StatusOK {
				err = errors.Errorf("Got status %d when downloading %s", resp.StatusCode, downloadURL)
			} else {
				data, err = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}
	}
	return func(ctx context.Context) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return data, nil
	}
}

func verifyRef(ref string) error {
	if ref != "" && !strings.HasPrefix(ref, "heads/") && !strings.HasPrefix(ref, "tags/") {
		return errors.New("ref must have a 'heads/' or 'tags/' prefix")
	}
	return nil
}

// recursiveGetContents downloads an entire github tree using the Github get-contents API
// (https://developer.github.com/v3/repos/contents/#get-contents).
type recursiveGetContents struct {
	tree    tree.Tree
	project *project
	mu      sync.Mutex
	wg      sync.WaitGroup
	errors  chan error
}

// Downloads download an entire (sub)tree of a github project using the get-contents API.
// The API returns an entire directory with all the files and download URL links.
// The API is called recursively on all the directories, and download all the content of
// all the files using the download URL.
// Each recursive call is called in a goroutine, and each content download is called in
// a goroutine.
// The synchronization is done using mu, and waiting for all the goroutine to finish is
// done using wg.
func (gc *recursiveGetContents) download(ctx context.Context) error {
	gc.wg.Add(1)
	gc.check(gc.recursive(ctx, gc.project.path))
	gc.wg.Wait()

	select {
	case err := <-gc.errors:
		return err
	default:
		return nil
	}
}

// recursice is a single recursive get-contents call. Before a call to recursive, wg.Add(1) should be called.
func (gc *recursiveGetContents) recursive(ctx context.Context, root string) error {
	defer gc.wg.Done()
	log.Printf("Using Github get-content API for path %q", root)
	file, entries, _, err := gc.project.client.Repositories.GetContents(ctx, gc.project.owner, gc.project.repo, root, gc.opt())
	if err != nil {
		return errors.Wrap(err, "github get-contents")
	}

	// This API call may return entries or file, we check both cases.
	for _, entry := range entries {
		fullPath := entry.GetPath()
		fsPath := strings.TrimPrefix(fullPath, gc.project.path)

		switch entry.GetType() {
		case "dir": // A directory.
			gc.mu.Lock()
			err = gc.tree.AddDir(fsPath)
			gc.mu.Unlock()
			if err != nil {
				return errors.Wrapf(err, "adding %s", fsPath)
			}
			gc.wg.Add(1)
			go gc.check(gc.recursive(ctx, fullPath))
		case "file": // A file.
			gc.wg.Add(1)
			go gc.check(gc.downloadContent(ctx, fsPath, entry.GetSize(), entry.GetDownloadURL()))
		}
	}

	if file != nil {
		path := file.GetPath()
		path = strings.TrimPrefix(path, gc.project.path)
		gc.mu.Lock()
		err = gc.tree.AddFile(path, file.GetSize(), contentFetchLoader(file.GetContent))
		gc.mu.Unlock()
		if err != nil {
			return errors.Wrapf(err, "adding %s", path)
		}
	}
	return nil
}

// downloadContent downloads content of a single file. Before a call to recursive, wg.Add(1) should be called.
func (gc *recursiveGetContents) downloadContent(ctx context.Context, path string, size int, downloadURL string) error {
	defer gc.wg.Done()
	loader := gc.project.contentDownloadLoader(ctx, downloadURL)
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.tree.AddFile(path, size, loader)
}

func contentFetchLoader(contentFetch func() (string, error)) func(ctx context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		content, err := contentFetch()
		if err != nil {
			return nil, err
		}
		return []byte(content), nil
	}
}

// opt returns Github GetContent options. The expected ref, unlike other APIs, should not
// have a 'heads/' or 'tags/' prefix.
func (gc *recursiveGetContents) opt() *github.RepositoryContentGetOptions {
	if gc.project.ref == "" {
		return nil
	}
	ref := strings.TrimPrefix(gc.project.ref, "heads/")
	ref = strings.TrimPrefix(ref, "tags/")
	return &github.RepositoryContentGetOptions{Ref: ref}
}

func (gc *recursiveGetContents) check(err error) {
	if err != nil {
		select {
		case gc.errors <- err:
		default:
			log.Printf("Failed sending error in channel", err)
		}
	}
}
