package application

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ProjectRunner struct{}

func NewProjectRunner() *ProjectRunner {
	return &ProjectRunner{}
}

// Run installs, builds if needed, then starts the project.
// it streams log lines into the provided send func.
func (pr *ProjectRunner) Run(
	ctx context.Context,
	path string,
	port int,
	send func(stream, text string),
) error {

	env := append(os.Environ(), fmt.Sprintf("PORT=%d", port))

	// ── detect framework ─────────────────────────────────────────────
	isNext := fileExists(filepath.Join(path, "next.config.js")) ||
		fileExists(filepath.Join(path, "next.config.ts")) ||
		fileExists(filepath.Join(path, "next.config.mjs")) ||
		hasDependency(path, "next")

	// ── npm install ──────────────────────────────────────────────────
	send("stdout", "[runner] running npm install...")
	if err := pr.runBuffered(ctx, path, env, send, "npm", "i", "--no-audit", "--no-fund"); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}
	send("stdout", "[runner] npm install complete")

	// ── build step (Next.js only for now) ───────────────────────────
	if isNext {
		send("stdout", "[runner] Next.js detected — running npm run build...")
		if err := pr.runBuffered(ctx, path, env, send, "npm", "run", "build"); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		send("stdout", "[runner] build complete")
	}

	// ── start ────────────────────────────────────────────────────────
	startCmd := pr.resolveStartCommand(ctx, path, env)
	send("stdout", fmt.Sprintf("[runner] starting on port %d...", port))
	return pr.runStreamed(ctx, path, env, send, startCmd)
}

// runBuffered runs a command and only streams output if it fails.
// good for install/build — you don't want 500 lines of npm output on success.
func (pr *ProjectRunner) runBuffered(
	ctx context.Context,
	dir string,
	env []string,
	send func(string, string),
	name string, args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		for _, line := range splitLines(buf.String()) {
			if line != "" {
				send("stderr", line)
			}
		}
		return err
	}
	return nil
}

// runStreamed runs a command and streams every line live.
// used for the actual app process.
func (pr *ProjectRunner) runStreamed(
	ctx context.Context,
	dir string,
	env []string,
	send func(string, string),
	cmd *exec.Cmd,
) error {
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	done := make(chan struct{}, 2)
	pipe := func(r interface{ Read([]byte) (int, error) }, stream string) {
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

	return cmd.Wait()
}

// resolveStartCommand picks the right start command for the project.
func (pr *ProjectRunner) resolveStartCommand(ctx context.Context, path string, env []string) *exec.Cmd {
	pkg := readPackageJSON(path)

	var cmd *exec.Cmd
	switch {
	case pkg.Scripts["start"] != "":
		cmd = exec.CommandContext(ctx, "npm", "start")
	case pkg.Scripts["dev"] != "":
		cmd = exec.CommandContext(ctx, "npm", "run", "dev")
	case pkg.Main != "":
		cmd = exec.CommandContext(ctx, "node", pkg.Main)
	default:
		cmd = exec.CommandContext(ctx, "node", "index.js")
	}

	cmd.Dir = path
	cmd.Env = env
	return cmd
}

// ── helpers ──────────────────────────────────────────────────────────────────

type packageJSON struct {
	Scripts      map[string]string `json:"scripts"`
	Main         string            `json:"main"`
	Dependencies map[string]string `json:"dependencies"`
}

// func readPackageJSON(path string) packageJSON {
// 	var pkg packageJSON
// 	data, err := os.ReadFile(filepath.Join(path, "package.json"))
// 	if err != nil {
// 		return pkg
// 	}
// 	fmt.Printf("PCKGDATA %v", data)
// 	json.Unmarshal(data, &pkg)
// 	return pkg
// }

func readPackageJSON(path string) packageJSON {
	var pkg packageJSON
	fullPath := filepath.Join(path, "package.json")
	fmt.Printf("READING PACKAGE JSON FROM: %s\n", fullPath)  // add this
	data, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Printf("ERROR READING: %v\n", err)  // add this
		return pkg
	}
	json.Unmarshal(data, &pkg)
	return pkg
}

func hasDependency(path, dep string) bool {
	pkg := readPackageJSON(path)
	fmt.Printf("PACKAGEJSON %v", pkg)
	_, ok := pkg.Dependencies[dep]
	return ok
}