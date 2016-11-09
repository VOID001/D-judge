package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

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
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, config.GlobalConfig.DockerVersion, nil, nil)
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

func (w *Worker) readExitCode(ctx context.Context) (code int, err error) {
	// Read the file exitcode and return

	path := filepath.Join(w.WorkDir, "exitcode")
	data, er := ioutil.ReadFile(path)
	if er != nil {
		err = errors.Wrap(er, "read exit code from file error")
		return
	}
	str := fmt.Sprintf("%s", data)
	str = strings.TrimSuffix(str, "\n")
	str = strings.TrimPrefix(str, "\n")
	code, err = strconv.Atoi(str)
	if err != nil {
		err = errors.Wrap(err, "read exit code from file error")
		return
	}
	return
}
