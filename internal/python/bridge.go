package python

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type Bridge struct {
	Interpreter string
	ScriptPath  string
}

func NewBridge(scriptPath string) (*Bridge, error) {
	interp, err := exec.LookPath("python3")
	if err != nil {
		interp, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("python interpreter not found: %w", err)
		}
	}

	if scriptPath == "" {
		return nil, fmt.Errorf("script path is required")
	}

	return &Bridge{
		Interpreter: interp,
		ScriptPath:  scriptPath,
	}, nil
}

func (b *Bridge) Process(data interface{}) (string, error) {
	input, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal input: %w", err)
	}

	cmd := exec.Command(b.Interpreter, b.ScriptPath)
	cmd.Stdin = strings.NewReader(string(input))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}

	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)

	err = cmd.Wait()
	if err != nil {
		if len(errBytes) > 0 {
			zap.L().Error("python stderr", zap.String("output", string(errBytes)))
		}
		return "", fmt.Errorf("python script failed: %w\nstderr: %s", err, string(errBytes))
	}

	if len(errBytes) > 0 {
		zap.L().Info("python stderr", zap.String("output", string(errBytes)))
	}

	return string(outBytes), nil
}
