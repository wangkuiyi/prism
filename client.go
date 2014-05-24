package prism

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"io"
	"log"
	"net/rpc"
	"path"
	"strings"
)

type Client struct {
	*rpc.Client
}

var (
	Addr = flag.String("prism", ":12340", "The listen address")
)

// Publish copies directory sourceDir to destDir.  It creates destDir
// if it does not exist yet.  Currently it does not copy
// sub-directories recursively.
func Publish(localDir, remotePath string) error {
	if (strings.HasPrefix(localDir, file.HDFSPrefix) ||
		strings.HasPrefix(remotePath, file.HDFSPrefix)) &&
		!file.IsConnectedToHDFS() {
		return fmt.Errorf("%s or %s on HDFS, but not connected",
			localDir, remotePath)
	}

	is, e := file.List(localDir)
	if e != nil {
		return fmt.Errorf("Cannot list source dir %s", localDir)
	}

	file.MkDir(path.Dir(remotePath))
	o, e := file.Create(remotePath)
	if e != nil {
		return fmt.Errorf("Create %s: %v", remotePath, e)
	}
	defer o.Close()

	w := zip.NewWriter(o)

	for _, i := range is {
		s := path.Join(localDir, i.Name)
		r, e := file.Open(s)
		if e != nil {
			return fmt.Errorf("Open %s: %v", s, e)
		}
		defer r.Close()

		o, e := w.Create(i.Name)
		if e != nil {
			return fmt.Errorf("Create zipped %s: %v", i.Name, e)
		}

		if _, e := io.Copy(o, r); e != nil {
			return fmt.Errorf("Zip %s/%s to %s: %v", s, i.Name, remotePath, e)
		}
	}

	if e := w.Close(); e != nil {
		return fmt.Errorf("Close zip archive %s: %v", remotePath, e)
	}

	return nil
}

// Connect connects to Prism server specified by command-line flag
// prism.Addr.
func Connect() (Client, error) {
	log.Printf("DialHTTP to Prism server %s", *Addr)
	c, e := rpc.DialHTTP("tcp", *Addr)
	if e != nil {
		return Client{nil}, fmt.Errorf(
			"Dialing Prism server %s error: %v", *Addr, e)
	}
	return Client{c}, nil
}

// Deploy downloads executables from remoteDir (usually HDFS) to localDir.
func (c Client) Deploy(remotePath, localDir string) error {
	if e := c.Call("Prism.Deploy",
		&Program{remotePath, localDir}, nil); e != nil {
		return fmt.Errorf("Prism.Deploy failed: %v", e)
	}
	return nil
}

// Launch launches a deployed executable, and assumes that the process
// will be listening on addr (indeed it does not have to).
func (c Client) Launch(addr, localDir, filename string,
	args []string, logbase string, retry int) error {
	e := c.Call("Prism.Launch",
		&Cmd{addr, localDir, filename, args, logbase, retry}, nil)
	if e != nil {
		return fmt.Errorf("Prism.Launch failed: %v", e)
	}
	return nil
}

func (c Client) Kill(addr string) error {
	if e := c.Call("Prism.Kill", addr, nil); e != nil {
		return fmt.Errorf("Prism.Kill(%s) failed: %v", addr, e)
	}
	return nil
}
