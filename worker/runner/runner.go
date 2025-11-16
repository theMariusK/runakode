package runner

import (
	"os"
	"context"
	"os/exec"
	"log"
	"fmt"
	"path/filepath"
	"time"
	"bytes"
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
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

	var codeFile string
	switch r.Language {
	case "python":
		codeFile = filepath.Join(tempDir, "main.py")
	case "go":
		codeFile = filepath.Join(tempDir, "main.go")
	default:
		codeFile = filepath.Join(tempDir, "main.py")
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
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

func Worker(id int, conn *amqp.Connection, jobs <-chan amqp.Delivery) {
	log.Printf("Worker (%d) started!\n", id)

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel on worker ID %d\n%v", id, err.Error())
		return
	}
	defer ch.Close()

	for job := range jobs {
		log.Printf("[Worker %d] Running a job (%d): %s\n", id, job.CorrelationId, job.Body)

		var request RunRequest
		err := json.Unmarshal([]byte(job.Body), &request)
		if err != nil {
			log.Println(err.Error())
			return
		}

		response := RunSandbox(&request)

		err = ch.Publish(
			"",
			job.ReplyTo,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				CorrelationId: job.CorrelationId,
				Body: []byte(response),
			},
		)
		if err != nil {
			log.Printf("[Worker %d] Publish failed!\n%v", id, err.Error())
		}
	}
}
