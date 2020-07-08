package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	root, err := os.Getwd()
	die(err)
	files := []string{
		"proto/rpc.pb.go",
		"proto/BUILD",
		"pkg/runtime/a_runtime-packr.go",
	}

	old := map[string][]byte{}
	for _, name := range files {
		b, err := ioutil.ReadFile(filepath.Join(root, name))
		if os.IsNotExist(err) {
			old[name] = nil
			continue
		}
		die(err)
		old[name] = b
	}
	cmd := exec.Command("bash", "hack/build_proto.sh")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	die(cmd.Run())

	cmd = exec.Command("packr")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	die(cmd.Run())
	for _, name := range files {
		b, err := ioutil.ReadFile(filepath.Join(root, name))
		die(err)
		if !bytes.Equal(b, old[name]) {
			fmt.Printf("file '%s' is changed, please commit again after adding files\n", name)
			fmt.Printf("\n")
			fmt.Printf("          git add %s", name)
			fmt.Printf("\n")
			os.Exit(1)
		}
	}
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}
