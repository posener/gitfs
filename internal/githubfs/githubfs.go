package githubfs

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
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

	if prefetch {
		t, err = fs.prefetchTree(ctx)
	} else {
		t, err = fs.getTree(ctx)
	}
	return t, err
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

// getTree gets a structure of a sub-tree of a github repository using the Github
// get-a-tree API: https://developer.github.com/v3/git/trees/#get-a-tree.
func (fs *githubfs) getTree(ctx context.Context) (tree.Tree, error) {
	gitTree, _, err := fs.client.Git.GetTree(ctx, fs.owner, fs.repo, fs.ref, true)
	if err != nil {
		return nil, errors.Wrap(err, "get git tree")
	}
	t := make(tree.Tree)
	for _, entry := range gitTree.Entries {
		path := entry.GetPath()
		if fs.path != "" {
			if !strings.HasPrefix(path, fs.path) {
				continue
			}
			path = strings.TrimPrefix(path, fs.path)
		}

		var err error
		switch entry.GetType() {
		case "tree": // A directory.
			if !fs.glob.Match(path, true) {
				continue
			}
			err = t.AddDir(path)
		case "blob": // A file.
			if !fs.glob.Match(path, false) {
				continue
			}
			err = t.AddFile(path, entry.GetSize(), fs.contentLoader(entry.GetSHA()))
		}
		if err != nil {
			return nil, errors.Wrapf(err, "adding %s", path)
		}
	}
	return t, nil
}

// prefetchTree download a tree from Github, with all of its files content.
func (fs *githubfs) prefetchTree(ctx context.Context) (tree.Tree, error) {
	downloader := recursiveGetContents{
		githubfs: fs,
		tree:     make(tree.Tree),
		errors:   make(chan error),
	}

	err := downloader.download(ctx)
	if err != nil {
		return nil, err
	}
	return downloader.tree, nil
}

// contentLoader gets content of git blob according to git sha of that blob.
func (fs *githubfs) contentLoader(sha string) func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		blob, _, err := fs.client.Git.GetBlob(ctx, fs.owner, fs.repo, sha)
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
func (fs *githubfs) contentDownloadLoader(ctx context.Context, downloadURL string) func(ctx context.Context) ([]byte, error) {
	var data []byte
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err == nil {
		resp, err := fs.httpClient.Do(req.WithContext(ctx))
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

// recursiveGetContents downloads an entire github tree using the Github get-contents API
// (https://developer.github.com/v3/repos/contents/#get-contents).
type recursiveGetContents struct {
	*githubfs
	tree   tree.Tree
	mu     sync.Mutex
	wg     sync.WaitGroup
	errors chan error
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
	gc.check(gc.recursive(ctx, gc.path))
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
	file, entries, _, err := gc.client.Repositories.GetContents(ctx, gc.owner, gc.repo, root, gc.opt())
	if err != nil {
		return errors.Wrap(err, "github get-contents")
	}

	// This API call may return entries or file, we check both cases.
	for _, entry := range entries {
		fullPath := entry.GetPath()
		fsPath := strings.TrimPrefix(fullPath, gc.path)

		switch entry.GetType() {
		case "dir": // A directory.
			if !gc.glob.Match(fsPath, true) {
				continue
			}
			gc.mu.Lock()
			err = gc.tree.AddDir(fsPath)
			gc.mu.Unlock()
			if err != nil {
				return errors.Wrapf(err, "adding %s", fsPath)
			}
			gc.wg.Add(1)
			go gc.check(gc.recursive(ctx, fullPath))
		case "file": // A file.
			if !gc.glob.Match(fsPath, false) {
				continue
			}
			gc.wg.Add(1)
			go gc.check(gc.downloadContent(ctx, fsPath, entry.GetSize(), entry.GetDownloadURL()))
		}
	}

	if file != nil {
		path := file.GetPath()
		path = strings.TrimPrefix(path, gc.path)
		if !gc.glob.Match(path, false) {
			return nil
		}
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
	loader := gc.contentDownloadLoader(ctx, downloadURL)
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
	if gc.ref == "" {
		return nil
	}
	ref := strings.TrimPrefix(gc.ref, "heads/")
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
