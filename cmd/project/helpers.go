package project

import (
	"log"
	"os"
	"os/exec"
	"strconv"
)

func passthruRun(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func passthruGitClone(localPath string, repoUrl string, depth int, branch string) error {
	args := []string{"git", "clone", repoUrl, localPath}
	if depth > 0 {
		args = append(args, "--depth="+strconv.Itoa(depth))
	}
	if branch != "" {
		args = append(args, "--branch="+branch)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Run Cmd >>> " + cmd.String())
	return cmd.Run()
}
