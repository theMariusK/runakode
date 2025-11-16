package runner

import (
	"os"
	"context"
	"os/exec"
	"log"
	"fmt"
	"path/filepath"
	"time"
	"encoding/json"
	"bytes"
)

type RunRequest struct {
	Language string `json:"language"`
	SourceCode string `json:"source_code"`
}

type RunResponse struct {
	Stdout string
	Stderr string
	ExitCode int
	Timeout bool
}

func RunSandbox(r *RunRequest) []byte {
	tempDir, err := os.MkdirTemp("", "sandbox-*")
	defer os.RemoveAll(tempDir)

	codeFile := filepath.Join(tempDir, "main.py")
	os.WriteFile(codeFile, []byte(r.SourceCode), 0644)

	image := fmt.Sprintf("%s-runner", r.Language)
	cmdArgs := []string{
		"run",
		"--rm",
		"--runtime=runsc",
		"--network=none",
		"-m", "128m",
		"--cpus", "0.5",
		"-v", tempDir + ":/sandbox:ro",
		image,
	}

	// TODO: time in config
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded

	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	response := RunResponse{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		ExitCode: exitCode,
		Timeout: timedOut,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Println(err.Error())
	}

	return jsonResponse
}
