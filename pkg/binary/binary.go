package binary

import (
	"context"
	"open-make-tiff/pkg/util"
	"os/exec"
)

type Config struct {
	Executable string
	Dir        string
	Args       []string
	Env        []string
}

type Binary struct {
	cfg Config
}

func New(cfg Config) *Binary {
	return &Binary{
		cfg: cfg,
	}
}

func (b *Binary) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, b.cfg.Executable, b.cfg.Args...)
	cmd.Env = b.cfg.Env
	cmd.Dir = b.cfg.Dir
	cmd.SysProcAttr = util.GetSysProcAttr()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return out, nil
}
