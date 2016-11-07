package controller

import (
	"context"
	"fmt"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/pkg/errors"
)

func (w *Worker) build(ctx context.Context) (err error) {
	// Start the container and Build the target
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Build error on Run#%d", w.JudgeInfo.SubmitID))
		return
	}

	//	log.Infof("MARK")
	cfg := container.Config{}
	cfg.Image = config.GlobalConfig.DockerImage
	cfg.WorkingDir = filepath.Join("/sandbox")
	cfg.User = "root" // Future will change to judge, a low-privileged user
	cfg.Tty = true
	cfg.AttachStdin = false
	cfg.AttachStderr = false
	cfg.AttachStdout = false
	cfg.Cmd = []string{"/bin/bash"}
	hcfg := container.HostConfig{}
	hcfg.Binds = []string{fmt.Sprintf("%s:%s", w.WorkDir, SandboxRoot)}
	log.Infof("Binds %s", fmt.Sprintf("%s:%s", w.WorkDir, SandboxRoot))
	hcfg.CpusetCpus = fmt.Sprintf("%d", w.CPUID)
	hcfg.Memory = config.GlobalConfig.RootMemory
	hcfg.PidsLimit = 64 // This is enough for almost all case

	resp, er := cli.ContainerCreate(ctx, &cfg, &hcfg, nil, "")
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Build error on Run#%d", w.JudgeInfo.SubmitID))
		return
	}
	log.Infof("MARK")
	defer cli.ContainerRemove(ctx, w.containerID, types.ContainerRemoveOptions{})
	w.containerID = resp.ID
	log.Debugf("RunID #%d container create ID %s", w.JudgeInfo.SubmitID, w.containerID)
	err = cli.ContainerStart(ctx, w.containerID, types.ContainerStartOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Build error on Run#%d", w.JudgeInfo.SubmitID))
		return
	}
	log.Infof("MARK")
	cmd := fmt.Sprintf("bash -c unzip -o build/%s -d build", w.JudgeInfo.BuildZip)
	log.Infof("container %s executing %s", w.containerID, cmd)
	info, err := w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, "Build error")
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Build error: RunID#%d exec command %+v return non-zero value %d", w.JudgeInfo.SubmitID, cmd, info.ExitCode))
		return
	}

	cmd = "bash -c build/build 2> build/build.err"
	log.Infof("container %s executing %s", w.containerID, cmd)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, "Build error")
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Build error: exec command %+v return non-zero value %d", cmd, info.ExitCode))
		return
	}
	// Do the real compile
	insp, err := cli.ContainerInspect(ctx, w.containerID)
	if err != nil {
		err = errors.Wrap(err, "Build error: inspect container")
		return
	}
	pid := insp.State.Pid
	cmd = fmt.Sprintf("bash -c build/run ./program DUMMY ./%s 2> ./compile.err > ./compile.out; touch ./done.lck", w.codeFileName)
	//cmd = fmt.Sprintf("./run ../program DUMMY ../%s", w.codeFileName)
	log.Debugf("container %s executing %s", w.containerID, cmd)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, "build error")
		return err
	}
	runinfo, er := w.runProtect(ctx, &insp, pid, uint64(30), w.JudgeInfo.OutputLimit, "")
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Build error on Run#%d", w.JudgeInfo.SubmitID))
		return
	}
	log.Infof("run protect exited, runinfo %+v", runinfo)
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("build error: exec command %+v return non-zero value %d", cmd, info.ExitCode))
		return
	}
	return
}
