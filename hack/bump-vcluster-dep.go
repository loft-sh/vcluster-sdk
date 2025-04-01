package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	workflowRelativePath = ".github/workflows/e2e.yaml"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("please pass target update version (vX.Y.Z, e.g. v0.24.0) as a first argument")
	}
	version := os.Args[1]
	if !strings.HasPrefix(version, "v") {
		log.Fatal("please pass target update version (vX.Y.Z, e.g. v0.24.0) with 'v' prefix")
	}
	workflowBytes, err := os.ReadFile(workflowRelativePath)
	if err != nil {
		log.Fatal(err)
	}
	var output string

	for _, line := range strings.Split(string(workflowBytes), "\n") {
		if strings.Contains(line, "  VCLUSTER_VERSION:") {
			replacedLine := "  VCLUSTER_VERSION: v" + version + "\n"
			log.Printf("replacing line: \n%s\n with: \n%s\n", line, replacedLine)
			output = output + replacedLine
		} else {
			output += line + "\n"
		}
	}
	if err := os.WriteFile(workflowRelativePath, []byte(output), 0644); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "get", "github.com/loft-sh/vcluster@v"+version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command("go", "mod", "tidy")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command("go", "mod", "vendor")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
