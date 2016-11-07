package controller

import (
	"context"

	"github.com/pkg/errors"

	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
)

type runinfo struct {
	usedmem      uint64
	usedtime     int64
	outputexceed bool
	timeexceed   bool
	memexceed    bool
}

type Worker struct {
	JudgeInfo    config.JudgeInfo
	WorkDir      string
	DockerImage  string
	RunUser      string
	CPUID        int
	MaxRetryTime int
	containerID  string
	codeFileName string
}

const (
	FilePerm    = 0644
	DirPerm     = 0755
	SandboxRoot = "/sandbox"
)

func (w *Worker) cleanup(ctx context.Context) (err error) {
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, "worker cleanup error")
		return err
	}
	err = cli.ContainerStop(ctx, w.containerID, nil)
	if err != nil {
		err = errors.Wrap(err, "worker cleanup error")
		return err
	}
	err = cli.ContainerRemove(ctx, w.containerID, types.ContainerRemoveOptions{})
	if err != nil {
		err = errors.Wrap(err, "worker cleanup error")
		return err
	}
	return
}
