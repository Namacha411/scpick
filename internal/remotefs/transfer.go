package remotefs

import (
	"fmt"
	"io"
	"os"
)

// Download streams remotePath to localPath, reporting progress via
// onProgress (which may be nil).
func (c *Client) Download(remotePath, localPath string, onProgress ProgressFunc) error {
	fi, err := c.sftp.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("remotefs: download %q: %w", remotePath, err)
	}

	src, err := c.sftp.Open(remotePath)
	if err != nil {
		return fmt.Errorf("remotefs: download %q: %w", remotePath, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("remotefs: download %q: %w", remotePath, err)
	}

	if err := copyWithProgress(dst, src, fi.Size(), onProgress); err != nil {
		_ = dst.Close()
		return fmt.Errorf("remotefs: download %q: %w", remotePath, err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("remotefs: download %q: %w", remotePath, err)
	}
	return nil
}

// Upload streams localPath to remotePath, reporting progress via
// onProgress (which may be nil).
func (c *Client) Upload(localPath, remotePath string, onProgress ProgressFunc) error {
	fi, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("remotefs: upload %q: %w", localPath, err)
	}

	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("remotefs: upload %q: %w", localPath, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := c.sftp.Create(remotePath)
	if err != nil {
		return fmt.Errorf("remotefs: upload %q: %w", remotePath, err)
	}

	if err := copyWithProgress(dst, src, fi.Size(), onProgress); err != nil {
		_ = dst.Close()
		return fmt.Errorf("remotefs: upload %q: %w", remotePath, err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("remotefs: upload %q: %w", remotePath, err)
	}
	return nil
}

// copyWithProgress streams src to dst, invoking onProgress after each chunk
// with the running byte count and the total size. It never buffers the
// whole file in memory, so large-file transfers stay flat on RAM.
func copyWithProgress(dst io.Writer, src io.Reader, total int64, onProgress ProgressFunc) error {
	if onProgress == nil {
		_, err := io.Copy(dst, src)
		return err
	}

	buf := make([]byte, 32*1024)
	var done int64
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			onProgress(done, total)
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}
