package githubfs

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/log"
	"github.com/posener/gitfs/internal/tree"
)

// getContents gets github content using Github's get-contents API:
// (https://developer.github.com/v3/repos/contents/#get-contents).
// It gets both the tree and the content of the files together.
type getContents githubfs

func (fs *getContents) get(ctx context.Context) (tree.Tree, error) {
	downloader := recursiveGetContents{
		getContents: fs,
		tree:        make(tree.Tree),
		errors:      make(chan error),
	}

	err := downloader.download(ctx)
	if err != nil {
		return nil, err
	}
	return downloader.tree, nil
}

// recursiveGetContents downloads an entire github tree using the Github get-contents API.
// Since this API returns only a single-depth level of files, it runs recursively on each
// directory. These recursive calls are done in parallel.
type recursiveGetContents struct {
	*getContents
	tree   tree.Tree
	mu     sync.Mutex
	wg     sync.WaitGroup
	errors chan error
}

// download an entire (sub)tree of a github project using the get-contents API.
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

// recursice is a single recursive get-contents call. Before a call to recursive,
// wg.Add(1) should be called.
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
		content, err := file.GetContent()
		if err != nil {
			return errors.Wrapf(err, "get content of %s", path)
		}
		gc.mu.Lock()
		err = gc.tree.AddFileContent(path, []byte(content))
		gc.mu.Unlock()
		if err != nil {
			return errors.Wrapf(err, "adding %s", path)
		}
	}
	return nil
}

// downloadContent downloads content of a single file. Before a call to recursive,
// wg.Add(1) should be called.
func (gc *recursiveGetContents) downloadContent(ctx context.Context, path string, size int, downloadURL string) error {
	defer gc.wg.Done()
	content, err := gc.downloadURL(ctx, downloadURL)
	if err != nil {
		return errors.Wrapf(err, "get content from %s", downloadURL)
	}
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.tree.AddFileContent(path, content)
}

// downloadContent downloads a given URL.
func (gc *recursiveGetContents) downloadURL(ctx context.Context, downloadURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "building request")
	}
	resp, err := gc.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "performing http request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got status %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
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
