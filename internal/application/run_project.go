package application

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"gowsrunner/internal/domain/entities"
	"gowsrunner/internal/domain/repository"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

// port counter — starts at 3000, increments per project
var portCounter atomic.Int32

func init() {
	portCounter.Store(3000)
}

func nextPort() int {
	return int(portCounter.Add(1))
}

// LogLine is what gets pushed over SSE
type LogLine struct {
	Stream string // "stdout" | "stderr"
	Text   string
}

type RunProjectUseCase struct {
	ProjectRepo repository.ProjectRepository
}

func NewRunProjectUseCase(repo repository.ProjectRepository) *RunProjectUseCase {
	return &RunProjectUseCase{ProjectRepo: repo}
}

// Execute starts the project and streams log lines into the returned channel.
// The channel is closed when the process exits or ctx is cancelled.
func (uc *RunProjectUseCase) Execute(
	ctx context.Context,
	projectID string,
) (<-chan LogLine, error) {

	project, err := uc.ProjectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if project.Status == entities.StatusRunning {
		return nil, fmt.Errorf("project is already running on port %d", project.Port)
	}

	port := nextPort()

	if err := uc.ProjectRepo.UpdatePortAndStatus(ctx, projectID, port, entities.StatusBuilding); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	logCh := make(chan LogLine, 64)

	go func() {
		defer close(logCh)

		send := func(stream, text string) {
			select {
			case logCh <- LogLine{Stream: stream, Text: text}:
			case <-ctx.Done():
			}
		}

		path := project.SourceLocation

		// ── npm install ──────────────────────────────────────────────
		send("stdout", fmt.Sprintf("[runner] npm install in %s", path))

		installCmd := exec.CommandContext(ctx, "npm", "i", "--no-audit", "--no-fund")
		installCmd.Dir = path
		installCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))

		var installBuf bytes.Buffer
		installCmd.Stdout = &installBuf
		installCmd.Stderr = &installBuf

		if err := installCmd.Run(); err != nil {
			for _, line := range splitLines(installBuf.String()) {
				send("stderr", line)
			}
			send("stderr", "[runner] npm install failed — aborting")
			_ = uc.ProjectRepo.UpdateStatus(ctx, projectID, entities.StatusFailed)
			return
		}
		send("stdout", "[runner] npm install done")

		// ── npm start / node server.js ───────────────────────────────
		hasPackageJSON := fileExists(filepath.Join(path, "package.json"))
		var runCmd *exec.Cmd
		if hasPackageJSON {
			runCmd = exec.CommandContext(ctx, "npm", "start")
		} else {
			runCmd = exec.CommandContext(ctx, "node", "server.js")
		}
		runCmd.Dir = path
		runCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))

		stdout, _ := runCmd.StdoutPipe()
		stderr, _ := runCmd.StderrPipe()

		if err := runCmd.Start(); err != nil {
			send("stderr", fmt.Sprintf("[runner] failed to start process: %v", err))
			_ = uc.ProjectRepo.UpdateStatus(ctx, projectID, entities.StatusFailed)
			return
		}

		_ = uc.ProjectRepo.UpdatePortAndStatus(ctx, projectID, port, entities.StatusRunning)
		send("stdout", fmt.Sprintf("[runner] process started on port %d (pid %d)", port, runCmd.Process.Pid))

		// stream stdout + stderr concurrently
		done := make(chan struct{}, 2)
		pipe := func(r io.Reader, stream string) {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				send(stream, sc.Text())
			}
			done <- struct{}{}
		}
		go pipe(stdout, "stdout")
		go pipe(stderr, "stderr")

		<-done
		<-done

		if err := runCmd.Wait(); err != nil {
			send("stderr", fmt.Sprintf("[runner] process exited: %v", err))
			_ = uc.ProjectRepo.UpdateStatus(context.Background(), projectID, entities.StatusFailed)
		} else {
			send("stdout", "[runner] process exited cleanly")
			_ = uc.ProjectRepo.UpdateStatus(context.Background(), projectID, entities.StatusStopped)
		}

		log.Printf("[runner] project %s finished", projectID)
	}()

	return logCh, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}