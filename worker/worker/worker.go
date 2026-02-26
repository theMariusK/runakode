package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/theMariusK/runakode/config"
)

type RunRequest struct {
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

type RunResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Timeout    bool   `json:"timeout"`
	ExecTimeMs int64  `json:"exec_time_ms"`
}

// limitedBuffer is a bytes.Buffer that caps writes at a maximum size.
type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (lb *limitedBuffer) Write(p []byte) (int, error) {
	remaining := lb.max - lb.buf.Len()
	if remaining <= 0 {
		return len(p), nil // discard but report success
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	return lb.buf.Write(p)
}

func (lb *limitedBuffer) String() string {
	return lb.buf.String()
}

func errorResponse(msg string) []byte {
	resp := RunResponse{
		Stderr:   msg,
		ExitCode: -1,
	}
	data, _ := json.Marshal(resp)
	return data
}

func RunSandbox(r *RunRequest, conf *config.Config) []byte {
	start := time.Now()

	tempDir, err := os.MkdirTemp("", "sandbox-*")
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to create temp dir: %v", err))
	}
	defer os.RemoveAll(tempDir)

	var codeFile string
	switch r.Language {
	case "python":
		codeFile = filepath.Join(tempDir, "main.py")
	case "go":
		codeFile = filepath.Join(tempDir, "main.go")
	default:
		return errorResponse(fmt.Sprintf("unsupported language: %s", r.Language))
	}

	if err := os.WriteFile(codeFile, []byte(r.SourceCode), 0644); err != nil {
		return errorResponse(fmt.Sprintf("failed to write code file: %v", err))
	}

	image, ok := conf.SandboxImages[r.Language]
	if !ok {
		return errorResponse(fmt.Sprintf("no sandbox image configured for language: %s", r.Language))
	}

	cmdArgs := []string{
		"run",
		"--rm",
		"--runtime=runsc",
		"--network=none",
		"-m", conf.JobMemory,
		"--cpus", conf.JobCPU,
		"-v", tempDir + ":/sandbox:ro",
		image,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.JobTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	stdout := &limitedBuffer{max: conf.OutputMaxBytes}
	stderr := &limitedBuffer{max: conf.OutputMaxBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		Timeout:    timedOut,
		ExecTimeMs: time.Since(start).Milliseconds(),
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		return errorResponse("failed to marshal response")
	}

	return jsonResponse
}

func Worker(id int, conn *amqp.Connection, jobs <-chan amqp.Delivery, conf *config.Config) {
	slog.Info("worker started", "worker_id", id)

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open channel", "worker_id", id, "error", err)
		return
	}
	defer ch.Close()

	for job := range jobs {
		slog.Info("running job", "worker_id", id, "correlation_id", job.CorrelationId, "body", string(job.Body))

		var request RunRequest
		err := json.Unmarshal(job.Body, &request)
		if err != nil {
			slog.Error("failed to unmarshal job", "worker_id", id, "error", err)
			continue
		}

		response := RunSandbox(&request, conf)

		err = ch.Publish(
			"",
			job.ReplyTo,
			false,
			false,
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: job.CorrelationId,
				Body:          response,
			},
		)
		if err != nil {
			slog.Error("publish failed", "worker_id", id, "error", err)
		}
	}
}
