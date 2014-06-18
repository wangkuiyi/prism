package prism

import (
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func TestFindDarwinProcess(t *testing.T) {
	gopath := os.Getenv("GOPATH")
	if len(gopath) <= 0 {
		t.Fatal("GOPATH environment variable is required for testing")
	}

	hello := path.Join(gopath, "bin", "hello")
	port := "18080"
	c := exec.Command(hello, "-addr=:"+port)
	if e := c.Start(); e != nil {
		t.Fatalf("Cannot start process hello: %v", e)
	}

	time.Sleep(500 * time.Millisecond)
	if e := findAndKillDarwinProcess(port); e != nil {
		t.Errorf("Failed findAndKillDarwinProcess(%s): %v", port, e)
	}
}
