package installer

import (
	"go.uber.org/zap"
	"os/exec"
)

type Executor struct {
	logger *zap.Logger
}

func NewExecutor(logger *zap.Logger) *Executor {
	return &Executor{logger: logger}
}

func (e *Executor) Run(name string, args ...string) error {
	e.logger.Info("exec", zap.String("name", name), zap.Strings("args", args))
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	e.logger.Info("exec output", zap.String("out", string(out)))
	return err
}
