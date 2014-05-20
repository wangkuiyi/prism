package prism

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
)

type Prism struct{}

// Deployment specifies to deploy Filename from RemoteDir to LocalDir.
// Both RemoteDir and LocalDir must have filesystem prefixes like
// "hdfs:" or "file:".
type Deployment struct {
	RemoteDir, LocalDir, Filename string
}

//------------------------------------------------------------------------------
func (p *Prism) Launch(d *Deployment, _ *int) error {
	remoteFile := path.Join(d.RemoteDir, d.Filename)
	localFile := path.Join(d.LocalDir, d.Filename)
	tempFile := fmt.Sprintf("%s.%d-%d",
		path.Join(d.LocalDir, d.Filename), os.Getpid(), rand.Int())

	r, e := file.Open(remoteFile)
	if e != nil {
		return fmt.Errorf("Cannot open HDFS file %s: %v", remoteFile, e)
	}
	defer r.Close()

	b, e := file.Exists(localFile)
	if e != nil {
		return fmt.Errorf("Cannot test existence of %s: %v", localFile, e)
	}

	if b { // If localFile already exists, compare MD5sum.
		w, e := file.Create(tempFile)
		if e != nil {
			return fmt.Errorf("Cannot create %s: %v", tempFile, e)
		}
		defer w.Close()

		side := io.TeeReader(r, w)
		sumRemote := make([]byte, 0)
		sumLocal := make([]byte, 0)
		if e := parallel.Do(
			func(sum []byte) error {
				h := md5.New()
				if _, e := io.Copy(h, side); e != nil {
					return fmt.Errorf("Error downloading %s or computing MD5: %v",
						remoteFile, e)
				}
				sum = h.Sum(sum)
				return nil
			}(sumRemote),
			func(sum []byte) error {
				h := md5.New()
				l, e := file.Open(localFile)
				if e != nil {
					return fmt.Errorf("Cannot open existing  %s: %v",
						localFile, e)
				}
				if _, e := io.Copy(h, l); e != nil {
					return fmt.Errorf("Error computing MD5 of %s: %v", localFile, e)
				}
				sum = h.Sum(sum)
				return nil
			}(sumLocal),
		); e != nil {
			return e
		}

		if !bytes.Equal(sumRemote, sumLocal) {
			if e := os.Rename(
				strings.TrimPrefix(tempFile, file.LocalPrefix),
				strings.TrimPrefix(localFile, file.LocalPrefix)); e != nil {
				return fmt.Errorf("Cannot rename %s to %s", tempFile, localFile)
			}
		} else {
			e := os.Remove(strings.TrimPrefix(tempFile, file.LocalPrefix))
			if e != nil {
				return fmt.Errorf("Cannot remove %s: %v", tempFile, e)
			}
		}

	} else { // local file does not exist
		w, e := file.Create(localFile)
		if e != nil {
			return fmt.Errorf("Cannot create %s: %v", localFile, e)
		}
		defer w.Close()

		if _, e := io.Copy(w, r); e != nil {
			return fmt.Errorf("Failed copying %s to %s: %v", remoteFile, localFile, e)
		}
	}

	return nil
}
