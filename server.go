package prism

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
)

type Prism struct{}

// Deployment specifies to deploy Filename from RemoteDir to LocalDir.
// Both RemoteDir and LocalDir must have filesystem prefixes like
// "hdfs:" or "file:".  When used with RPC DeployFile, Filename must
// not be nil or empty, and DeployFile will copy RemoteDir/Filename to
// LocalDir/Filename.  When used with RPC DeployDir, Filename is
// ignored, and all files in RemoteDir are copied to LocalDir.
type Deployment struct {
	RemoteDir, LocalDir, Filename string
}

// Command specifies the command as well as Args and LogBase that are
// used to start a local process.  LocalDir/Filename must exists.
// LogBase is a local directory which hold log files whose content
// include the standard outputs and error of the launched process.
// Command is supposed to be used with RPC Launch.
type Command struct {
	LocalDir, Filename string
	Args               []string
	LogBase            string
	Retry              int
}

func (p *Prism) DeployFile(d *Deployment, _ *int) error {
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
			func() error {
				h := md5.New()
				if _, e := io.Copy(h, side); e != nil {
					return fmt.Errorf("Error downloading %s or computing MD5: %v",
						remoteFile, e)
				}
				sumRemote = h.Sum(sumRemote)
				return nil
			},
			func() error {
				h := md5.New()
				l, e := file.Open(localFile)
				if e != nil {
					return fmt.Errorf("Cannot open existing  %s: %v",
						localFile, e)
				}
				if _, e := io.Copy(h, l); e != nil {
					return fmt.Errorf("Error computing MD5 of %s: %v", localFile, e)
				}
				sumLocal = h.Sum(sumRemote)
				return nil
			},
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

func (p *Prism) Launch(cmd *Command, _ *int) error {
	aggregateErrors := func(es ...error) error {
		r := ""
		for _, e := range es {
			if e != nil {
				r += fmt.Sprintf("%v\n", e)
			}
		}
		if r != "" {
			return errors.New(r)
		}
		return nil
	}

	exe := path.Join(strings.TrimPrefix(cmd.LocalDir, file.LocalPrefix), cmd.Filename)
	logfile := path.Join(strings.TrimPrefix(cmd.LogBase, file.LocalPrefix), cmd.Filename)
	c := exec.Command(exe, cmd.Args...)
	fout, e1 := os.Create(logfile + ".out")
	ferr, e2 := os.Create(logfile + ".err")
	cout, e3 := c.StdoutPipe()
	cerr, e4 := c.StderrPipe()
	if e := aggregateErrors(e1, e2, e3, e4); e != nil {
		return e
	}
	go io.Copy(fout, cout)
	go io.Copy(ferr, cerr)
	go func(c *exec.Cmd, cmd Command) {
		for i := 0; i < cmd.Retry; i++ {
			log.Printf("Start process %v", cmd)
			if e := c.Run(); e != nil {
				log.Printf("Restart process %v: %v", cmd, e)
			} else {
				log.Printf("%v successfully finished.", cmd)
				break
			}
		}
		log.Printf("No more restart of %v.", cmd)
	}(c, *cmd)

	return nil
}
