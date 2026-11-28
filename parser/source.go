package parser

import (
	"errors"
	"io/fs"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

var (
	ErrNotSourceFS = errors.New("parser: type does not implement SourceFS interface")
)

// Source is an interface for a filesystem that can be used as a source for a Parser.
type SourceFS interface {
	// Get the path to copy a file to at the destination filesystem.
	// Use filePath at sourceFS to determine the path the file should be copied to at the destination filesystem.
	GetDestinationFilePath(filePath string) string

	// Get the path to copy a directory to at the destination filesystem.
	// Use dirPath at sourceFS to determine the path the directory should be copied to at the destination filesystem.
	GetDestinationDirPath(dirPath string) string
}

// Get the path to copy a file to at the destination filesystem.
// Use filePath at sourceFS to determine the path the file should be copied to at the destination filesystem.
//
// An error will be returned if source is not a SourceFS.
func GetDestinationFilePath(source fs.FS, filePath string) (string, error) {
	if source, ok := source.(SourceFS); ok {
		return source.GetDestinationFilePath(filePath), nil
	}
	return "", ErrNotSourceFS
}

// Get the path to copy a directory to at the destination filesystem.
// Use dirPath at sourceFS to determine the path the directory should be copied to at the destination filesystem.
//
// An error will be returned if source is not a SourceFS.
func GetDestinationDirPath(source fs.FS, dirPath string) (string, error) {
	if source, ok := source.(SourceFS); ok {
		return source.GetDestinationDirPath(dirPath), nil
	}
	return "", ErrNotSourceFS
}

// A clone of a git repo: its files, the directory they are in and the commit.
type gitClone struct {
	fsys   fs.FS
	dir    string
	commit string
}

// Create a new source filesystem from a git repo at url.
func newGitFSWithAuth(url string, branch string, auth transport.AuthMethod) (gitClone, error) {
	dir, err := mkdirTemp()
	if err != nil {
		return gitClone{}, err
	}

	opts := &git.CloneOptions{
		URL:   url,
		Auth:  auth,
		Depth: 1,
	}

	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}

	repo, err := git.PlainClone(dir, false, opts)
	if err != nil {
		_ = os.RemoveAll(dir)
		return gitClone{}, err
	}

	ref, err := repo.Head()
	if err != nil {
		_ = os.RemoveAll(dir)
		return gitClone{}, err
	}

	log.Println("cloned", url, "at", ref.Hash().String())

	return gitClone{fsys: os.DirFS(dir), dir: dir, commit: ref.Hash().String()}, nil
}

// CloseSource removes the files of a source that was cloned for parsing (a git repo); other sources are left alone.
func CloseSource(source fs.FS) error {
	switch s := source.(type) {
	case LabRepoSource:
		if s.Dir != "" {
			return os.RemoveAll(s.Dir)
		}
	case DeviceRepoSource:
		if s.dir != "" {
			return os.RemoveAll(s.dir)
		}
	}
	return nil
}

// SourceCommit returns the commit of a git source, or "" for a directory.
func SourceCommit(source fs.FS) string {
	if repo, ok := source.(LabRepoSource); ok {
		return repo.Commit
	}
	return ""
}

// Create a new temporary directory.
func mkdirTemp() (string, error) {
	return os.MkdirTemp("", "needforheat-manual-server-git_*")
}
