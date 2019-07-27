package githubfs

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/tree"
)

// getATree gets github tree using Github's get-a-tree API:
// https://developer.github.com/v3/git/trees/#get-a-tree.
// The content provider returns the file content only when accessed.
type getATree githubfs

func (fs *getATree) get(ctx context.Context) (tree.Tree, error) {
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

// contentLoader gets content of git blob according to git sha of that blob.
func (fs *getATree) contentLoader(sha string) func(context.Context) ([]byte, error) {
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
