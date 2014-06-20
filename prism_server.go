package prism

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/parallel"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type Prism struct {
	mutex     sync.Mutex
	notifiers map[string]chan bool
}

func NewPrism() *Prism {
	return &Prism{notifiers: make(map[string]chan bool)}
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
					return fmt.Errorf("Download %s or computing MD5: %v",
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
					return fmt.Errorf("Compute MD5 of %s: %v", localFile, e)
				}
				sumLocal = h.Sum(sumLocal)
				return nil
			},
		); e != nil {
			return e
		}

		if !bytes.Equal(sumRemote, sumLocal) {
			if e := os.Rename(
				strings.TrimPrefix(tempFile, file.LocalPrefix),
				strings.TrimPrefix(localFile, file.LocalPrefix)); e != nil {
				return fmt.Errorf("Rename %s to %s", tempFile, localFile)
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
	// We do not check return value since it is just a safe-to-do.
	p.Kill(cmd.Addr, nil)

	exe := path.Join(strings.TrimPrefix(cmd.LocalDir, file.LocalPrefix),
		cmd.Filename)
	os.Chmod(exe, 0774)

	file.MkDir(cmd.LogDir)
	logfile := path.Join(cmd.LogDir, cmd.Filename+"-"+cmd.Addr)

	log.Printf("Prism launches %s %v on %s with retry=%d",
		exe, cmd.Args, cmd.Addr, cmd.Retry)
	go func(cmd Cmd) {
		if _, exist := p.notifiers[cmd.Addr]; exist {
			log.Printf("Cannot start %s, which is already started.", cmd.Addr)
			return
		} else {
			p.mutex.Lock()
			p.notifiers[cmd.Addr] = make(chan bool, 1)
			p.mutex.Unlock()
			defer func() {
				p.mutex.Lock()
				defer p.mutex.Unlock()
				delete(p.notifiers, cmd.Addr)
			}()
		}

		fout, e1 := file.Create(logfile + ".out")
		ferr, e2 := file.Create(logfile + ".err")
		if e1 != nil || e2 != nil {
			log.Print("Prism failed create log files: %v %v", e1, e2)
			return
		}
		defer fout.Close()
		defer ferr.Close()

		for i := 0; i < cmd.Retry; i++ {
			select {
			case <-p.notifiers[cmd.Addr]:
				log.Printf("%v killed intensionally. No restart", cmd.Addr)
				return
			default:
			}

			c := exec.Command(exe, cmd.Args...)
			cout, e1 := c.StdoutPipe()
			cerr, e2 := c.StderrPipe()
			if e1 != nil || e2 != nil {
				log.Print("Prism failed retrieve cmd pipes: %v %v", e1, e2)
				return
			}
			go io.Copy(fout, cout)
			go io.Copy(ferr, cerr)

			if e := c.Run(); e != nil {
				log.Printf("Prism's launch %s stopped: %v", cmd.Addr, e)
			} else {
				log.Printf("Prism's launch %s completed.", cmd.Addr)
				break
			}
		}
		log.Printf("No more restart of %s.", cmd.Addr)
	}(*cmd)

	return nil
}

// Kill kills a process that was started by Launch and is listening on addr.
func (p *Prism) Kill(addr string, _ *int) error {
	// Close notifier channel to prevent Prism from restarting the
	// process in case Retry > 1.
	p.mutex.Lock()
	notifier, exists := p.notifiers[addr]
	if exists {
		close(notifier)
		// If !exists, the process might still exist if it was started
		// by another starter.
	}
	p.mutex.Unlock()

	f := strings.Split(addr, ":")
	if runtime.GOOS == "linux" {
		o, e := exec.Command("fuser", "-k", "-n", "tcp", f[1]).CombinedOutput()
		if e != nil {
			return fmt.Errorf("fuser %s failed: %v, with output %s",
				f[1], e, o)
		}
	} else if runtime.GOOS == "darwin" {
		if e := findAndKillDarwinProcess(f[1]); e != nil {
			return fmt.Errorf("Failed to kill %s: %v", f[1], e)
		}
	}
	return nil
}

func KillAll(p *Prism) error {
	e := parallel.RangeMap(p.notifiers, func(k, _ reflect.Value) error {
		return p.Kill(k.String(), nil)
	})
	p.notifiers = make(map[string]chan bool) // Reset notifiers.
	return e
}

func findAndKillDarwinProcess(port string) error {
	o, e := exec.Command("lsof", "-V", "-i:"+port).CombinedOutput()
	if e != nil {
		return fmt.Errorf("lsof -i%s failed: %v, with output %s", port, e, o)
	}

	s := bufio.NewScanner(bytes.NewReader(o))
	pid := ""
	for s.Scan() {
		if strings.Contains(s.Text(), "LISTEN") {
			if pid == "" {
				pid = strings.Fields(s.Text())[1]
			} else {
				return fmt.Errorf("More than one LISTENing processes: %s", o)
			}
		}
	}

	if len(pid) == 0 {
		return fmt.Errorf("Found no process: %s", o)
	}

	o, e = exec.Command("kill", "-KILL", pid).CombinedOutput()
	if e != nil {
		return fmt.Errorf("kill %s failed: %v, with output %s", pid, e, o)
	}

	return nil
}
