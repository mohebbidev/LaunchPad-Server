package application



type StartNodeApplication struct {
	
}

package service

import (
	"bytes"
	"fmt"
	"gowsrunner/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type ProjectService struct{}

// func (service *ProjectService) StartNodeApplication(path string, port int) *exec.Cmd {
// 	hasPackageJSON := utils.FileExists(filepath.Join(path, "package.json"))
// 	env := append(os.Environ(), fmt.Sprintf("PORT=%d", port))

// 	var cmd *exec.Cmd

// 	installCmd := exec.Command("npm", "i", "--no-audit", "--no-fund")
// 	installCmd.Dir = path
// 	installCmd.Env = env

// 	var installBuf bytes.Buffer
// 	installCmd.Stderr = &installBuf
// 	installCmd.Stdout = &installBuf

// 	log.Printf("running npm install in %v", path)
// 	if err := installCmd.Run(); err != nil {
// 		log.Printf("npm install failed: %s", installBuf.String())
// 		// return a process that fails fast
// 		cmd = exec.Command("bash", "-c", "echo npm install failed; exit 1")
// 		cmd.Dir = path
// 		cmd.Env = env
// 		return cmd
// 	}

// 	if hasPackageJSON {
// 		cmd = exec.Command("npm", "start")
// 	} else {
// 		// fallback
// 		cmd = exec.Command("node", "server.js")
// 	}

// 	cmd.Dir = path
// 	cmd.Env = env

// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	if err := cmd.Start(); err != nil {
// 		log.Printf("failed to start: %v", err)
// 	}

// 	time.Sleep(800 * time.Millisecond)

// 	return cmd

// }
