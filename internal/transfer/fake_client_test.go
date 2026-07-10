package transfer

import (
	"fmt"

	"scpick/internal/remotefs"
)

// fakeClient implements the remoteFile interface for tests, without
// touching a real SSH/SFTP connection.
type fakeClient struct {
	stat     map[string]remotefs.Entry
	statErr  map[string]error
	listDir  func(path string) ([]remotefs.Entry, error)
	mkdirAll func(path string) error
	download func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error
	upload   func(localPath, remotePath string, onProgress remotefs.ProgressFunc) error
}

func (f *fakeClient) ListDir(path string) ([]remotefs.Entry, error) {
	if f.listDir != nil {
		return f.listDir(path)
	}
	return nil, fmt.Errorf("fakeClient: ListDir not used in this test")
}

func (f *fakeClient) MkdirAll(path string) error {
	if f.mkdirAll != nil {
		return f.mkdirAll(path)
	}
	return nil
}

func (f *fakeClient) Stat(path string) (remotefs.Entry, error) {
	if err, ok := f.statErr[path]; ok {
		return remotefs.Entry{}, err
	}
	if e, ok := f.stat[path]; ok {
		return e, nil
	}
	return remotefs.Entry{}, fmt.Errorf("fakeClient: stat %q: not found", path)
}

func (f *fakeClient) Download(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
	return f.download(remotePath, localPath, onProgress)
}

func (f *fakeClient) Upload(localPath, remotePath string, onProgress remotefs.ProgressFunc) error {
	return f.upload(localPath, remotePath, onProgress)
}
