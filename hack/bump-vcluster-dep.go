package main

import (
	"bufio"
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
	workflowFile, err := os.Open(workflowRelativePath)
	if err != nil {
		log.Fatal(err)
	}
	defer workflowFile.Close()

	versionWithoutV := strings.TrimPrefix(version, "v")

	scanner := bufio.NewScanner(workflowFile)
	var output string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "  VCLUSTER_VERSION:") {
			replacedLine := "  VCLUSTER_VERSION: " + version + "\n"
			log.Printf("replacing line: \n%s\n with: \n%s\n", line, replacedLine)
			output = output + replacedLine
		} else if strings.Contains(line, "  VCLUSTER_BACKGROUND_PROXY_IMAGE: ghcr.io/loft-sh/vcluster-pro:") {
			replacedLine := "  VCLUSTER_BACKGROUND_PROXY_IMAGE: ghcr.io/loft-sh/vcluster-pro:" + versionWithoutV + "\n"
			log.Printf("replacing line: \n%s\n with: \n%s\n", line, replacedLine)
			output = output + replacedLine
		} else {
			output += line + "\n"
		}
	}
	if err := os.WriteFile(workflowRelativePath, []byte(output), 0644); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "get", "github.com/loft-sh/vcluster@"+version)
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
