package prism

import (
	"archive/zip"
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
	"runtime"
	"strings"
	"time"
)

type Prism struct {
	notifiers map[string]chan bool
}

func NewPrism() *Prism {
	return &Prism{make(map[string]chan bool)}
}

// Program an deployment of RemotePath to LocalDir.  RemotePath must
// specify a zip file which contains a package of one or more files.
// Both RemotePath and LocalDir must have filesystem prefixes like
// "hdfs:" or "file:".
type Program struct {
	RemotePath, LocalDir string
}

// Cmd specifies the command as well as Args and LogDir that are
// used to start a local process.  LocalDir/Filename must exists.
// LogDir is a local directory which hold log files whose content
// include the standard outputs and error of the launched process.
// Cmd is supposed to be used with RPC Launch.
type Cmd struct {
	Addr               string
	LocalDir, Filename string
	Args               []string
	LogDir             string
	Retry              int
}

// Deploy unpack an zip file specified by Program.RemotePath to
// directory Program.LocalDir.
func (p *Prism) Deploy(d *Program, _ *int) error {
	localFile := path.Join(d.LocalDir, path.Base(d.RemotePath))
	tempFile := fmt.Sprintf("%s.%d-%d", localFile, os.Getpid(), rand.Int())

	r, e := file.Open(d.RemotePath)
	if e != nil {
		return fmt.Errorf("Cannot open HDFS file %s: %v", d.RemotePath, e)
	}
	defer r.Close()

	b, e := file.Exists(localFile)
	if e != nil {
		return fmt.Errorf("Cannot test existence of %s: %v", localFile, e)
	}

	if b { // If localFile already exists, compare MD5 checksum.
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
						d.RemotePath, e)
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
		file.MkDir(d.LocalDir)
		w, e := file.Create(localFile)
		if e != nil {
			return fmt.Errorf("Cannot create %s: %v", localFile, e)
		}
		defer w.Close()

		if _, e := io.Copy(w, r); e != nil {
			return fmt.Errorf("Failed copying %s to %s: %v",
				d.RemotePath, localFile, e)
		}
	}

	if strings.HasPrefix(localFile, file.LocalPrefix) {
		// Remove filesystem prefix, as unzipLocal handles only local file.
		fs1 := strings.Split(localFile, ":")
		fs2 := strings.Split(d.LocalDir, ":")
		unzipLocal(strings.Join(fs1[1:], ":"), strings.Join(fs2[1:], ":"))
	}

	return nil
}

func unzipLocal(name, dir string) error {
	r, e := zip.OpenReader(name)
	if e != nil {
		return fmt.Errorf("Cannot open %s: %v", name, e)
	}
	defer r.Close()

	for _, f := range r.File {
		rc, e := f.Open()
		if e != nil {
			return fmt.Errorf("Open zipped %s: %v", f.Name)
		}

		outFile := path.Join(dir, f.Name)
		o, e := os.Create(outFile)
		if e != nil {
			return fmt.Errorf("Cannot create %s: %v", outFile)
		}

		if _, e := io.Copy(o, rc); e != nil {
			return fmt.Errorf("Copy %s to %s: %v", f.Name, outFile, e)
		}

		o.Close()
		rc.Close()
	}

	return nil
}

func (p *Prism) Launch(cmd *Cmd, _ *int) error {
	if e := p.Kill(cmd.Addr, nil); e != nil {
		log.Printf("Kill %s before launch failed: %v", cmd.Addr, e)
	}

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

	exe := path.Join(strings.TrimPrefix(cmd.LocalDir, file.LocalPrefix),
		cmd.Filename)
	os.Chmod(exe, 0774)

	file.MkDir(cmd.LogDir)

	logfile := path.Join(cmd.LogDir, cmd.Filename+"-"+cmd.Addr)
	c := exec.Command(exe, cmd.Args...)
	fout, e1 := file.Create(logfile + ".out")
	ferr, e2 := file.Create(logfile + ".err")
	cout, e3 := c.StdoutPipe()
	cerr, e4 := c.StderrPipe()
	if e := aggregateErrors(e1, e2, e3, e4); e != nil {
		return e
	}
	go io.Copy(fout, cout)
	go io.Copy(ferr, cerr)

	log.Printf("Launch %s %v as %s", exe, cmd.Args, cmd.Addr)
	go func(c *exec.Cmd, cmd Cmd) {
		if _, exist := p.notifiers[cmd.Addr]; exist {
			log.Printf("Cannot start %s, which is already started.", cmd.Addr)
			return
		} else {
			p.notifiers[cmd.Addr] = make(chan bool, 1)
		}

		for i := 0; i < cmd.Retry; i++ {
			select {
			case <-p.notifiers[cmd.Addr]:
				log.Printf("%v being killed intensionally.", cmd.Addr)
				break
			default:
			}

			if e := c.Run(); e != nil {
				log.Printf("Launch %s failed: %v", cmd.Addr, e)
				time.Sleep(time.Second) // debug
			} else {
				log.Printf("%s successfully finished.", cmd.Addr)
				break
			}
		}

		delete(p.notifiers, cmd.Addr)
		log.Printf("No more restart of %s.", cmd.Addr)
	}(c, *cmd)

	return nil
}

// Kill kills a process that was started by Launch and is listening on addr.
func (p *Prism) Kill(addr string, _ *int) error {
	// Close notifier channel to prevent Prism from restarting the
	// process in case Retry > 1.
	notifier, exists := p.notifiers[addr]
	if exists {
		close(notifier)
		// If !exists, the process might still exist if it was started
		// by another starter.
	}

	f := strings.Split(addr, ":")
	if runtime.GOOS == "linux" {
		o, e := exec.Command("fuser", "-k", "-n", "tcp", f[1]).CombinedOutput()
		if e != nil {
			return fmt.Errorf("fuser %s failed: %v, with output %s", f[1], e, o)
		}
	} else if runtime.GOOS == "darwin" {
		o, e := exec.Command("lsof", "-i:"+f[1], "-t").CombinedOutput()
		if e != nil {
			return fmt.Errorf("lsof %s failed: %v, with output %s", f[1], e, o)
		}
		pid := strings.TrimSuffix(string(o), "\n")
		o, e = exec.Command("kill", "-KILL", pid).CombinedOutput()
		if e != nil {
			return fmt.Errorf("kill %s failed: %v, with output %s", pid, e, o)
		}
	}
	return nil
}
