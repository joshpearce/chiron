package devagent

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// Shell runs commands for real.
type Shell struct{}

func (Shell) Run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Stream runs a command and hands over each line of its stdout as it
// arrives; stderr goes to the agent's own log.
func (Shell) Stream(dir, name string, args []string, onLine func(string)) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			onLine(line)
		}
	}
	return cmd.Wait()
}
