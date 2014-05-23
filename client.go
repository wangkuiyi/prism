package prism

import (
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
func Publish(sourceDir, destDir string) error {
	if (strings.HasPrefix(sourceDir, file.HDFSPrefix) ||
		strings.HasPrefix(destDir, file.HDFSPrefix)) &&
		!file.IsConnectedToHDFS() {
		return fmt.Errorf("source (%s) or dest (%s) on HDFS, but not connected",
			sourceDir, destDir)
	}

	is, e := file.List(sourceDir)
	if e != nil {
		return fmt.Errorf("Cannot list source dir %s", sourceDir)
	}

	// Create the destination directory if not exists yet.
	file.MkDir(destDir)

	for _, i := range is {
		s := path.Join(sourceDir, i.Name)
		d := path.Join(destDir, i.Name)

		r, e := file.Open(s)
		if e != nil {
			return fmt.Errorf("Cannot open %s: %v", s, e)
		}
		defer r.Close()

		w, e := file.Create(d)
		if e != nil {
			return fmt.Errorf("Cannot create %s: %v", d, e)
		}
		defer w.Close()

		if _, e := io.Copy(w, r); e != nil {
			return fmt.Errorf("Failed copy from %s to %s: %v", s, d, e)
		}
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
func (c Client) Deploy(remoteDir, localDir, filename string) error {
	if e := c.Call("Prism.Deploy",
		&Program{remoteDir, localDir, filename}, nil); e != nil {
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
