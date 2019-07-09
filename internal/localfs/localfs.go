package localfs

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-billy.v4/osfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

// New returns a Tree for a given github project name.
func New(projectName string, localPath string) (http.FileSystem, error) {
	gitRoot, err := lookupGitRoot(localPath)
	if err != nil {
		return nil, errors.Wrap(err, "git root not found")
	}
	subDir, err := computeSubdir(projectName, gitRoot)
	if err != nil {
		return nil, errors.Wrap(err, "git repository does not match project")
	}
	return http.Dir(filepath.Join(gitRoot, subDir)), nil
}

// match validates tha the git repository has a remote URL that matches
// the given project.
func computeSubdir(projectName, gitRoot string) (string, error) {
	projectName = cleanRevision(projectName)
	r, err := gitRepo(gitRoot)
	if err != nil {
		return "", err
	}
	remotes, err := r.Remotes()
	if err != nil {
		return "", err
	}
	for _, remote := range remotes {
		for _, url := range remote.Config().URLs {
			project := urlProjectName(url)
			if projectName == project {
				return "", nil
			}
			if strings.HasPrefix(projectName, project+"/") {
				return strings.TrimPrefix(projectName, project+"/"), nil
			}
		}
	}
	return "", errors.New("non of remote URLs matched")
}

func cleanRevision(projectName string) string {
	i := strings.Index(projectName, "@")
	if i < 0 {
		return projectName
	}
	return projectName[:i]
}

func gitRepo(path string) (*git.Repository, error) {
	// We instantiate a new repository targeting the given path (the .git folder)
	fs := osfs.New(path)
	if _, err := fs.Stat(git.GitDirName); err == nil {
		fs, err = fs.Chroot(git.GitDirName)
		if err != nil {
			return nil, err
		}
	}

	s := filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})
	return git.Open(s, fs)
}

func lookupGitRoot(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for path != "" {
		if _, err := os.Stat(filepath.Join(path, git.GitDirName)); err == nil {
			return path, nil
		}
		path, _ = filepath.Split(path)
		if len(path) > 0 && path[len(path)-1] == filepath.Separator {
			path = path[:len(path)-1]
		}
	}
	return "", errors.New("not git repository")
}

func urlProjectName(urlStr string) string {
	url, err := url.Parse(urlStr)
	if err != nil {
		panic(fmt.Sprintf("failed parsing URL: %s", urlStr))
	}
	url.Path = strings.TrimSuffix(url.Path, ".git")
	return url.Host + url.Path
}
