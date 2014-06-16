package prism

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"io"
	"net/rpc"
	"path"
	"strings"
)

var (
	Port = flag.Int("prism_port", 12340, "Listening port of Prism")
)

// Publish packs localDir into a zip file named by remotePath.  It
// creates all necessary levels of parent directories of remotePath.
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

// Connect connects to the Prism server running on host and listening
// on Port.
func connect(host string) (*rpc.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, *Port)
	c, e := rpc.DialHTTP("tcp", addr)
	if e != nil {
		return nil, fmt.Errorf("Dialing Prism %s: %v", addr, e)
	}
	return c, nil
}

// Deploy downloads the zip archive remotePath, usually created by
// Publish and put on HDFS, to localDir of host.  localDir must
// exists.
func Deploy(host, remotePath, localDir string) error {
	c, e := connect(host)
	if e != nil {
		return e
	}
	defer c.Close()

	e = c.Call("Prism.Deploy", &Program{remotePath, localDir}, nil)
	if e != nil {
		return fmt.Errorf("Prism.Deploy failed: %v", e)
	}
	return nil
}

// Launch launches a deployed executable, and assumes that the process
// will be listening on addr (indeed it does not have to).
func Launch(addr, localDir, filename string,
	args []string, logDir string, retry int) error {

	fields := strings.Split(addr, ":")
	if len(fields) != 2 || len(fields[0]) <= 0 || len(fields[1]) <= 0 {
		return fmt.Errorf("Launch addr %s not in form of host:port")
	}

	c, e := connect(fields[0])
	if e != nil {
		return e
	}
	defer c.Close()

	e = c.Call("Prism.Launch",
		&Cmd{addr, localDir, filename, args, logDir, retry}, nil)
	if e != nil {
		return fmt.Errorf("Prism.Launch failed: %v", e)
	}
	return nil
}

func Kill(addr string) error {
	fields := strings.Split(addr, ":")
	if len(fields) != 2 || len(fields[0]) <= 0 || len(fields[1]) <= 0 {
		return fmt.Errorf("Launch addr %s not in form of host:port")
	}

	c, e := connect(fields[0])
	if e != nil {
		return e
	}
	defer c.Close()

	if e := c.Call("Prism.Kill", addr, nil); e != nil {
		return fmt.Errorf("Prism.Kill(%s) failed: %v", addr, e)
	}
	return nil
}
