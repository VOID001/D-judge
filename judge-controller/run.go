package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
)

func (w *Worker) run(ctx context.Context, tid int64) (err error) {
	// Prepare the run script
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	cmd := fmt.Sprintf("/bin/bash -c unzip -o run/%s -d run", w.JudgeInfo.RunZip)
	info, er := w.execcmd(ctx, cli, "root", cmd)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Run error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, tid, cmd, info.ExitCode))
		return
	}
	cmd = fmt.Sprintf("/bin/bash -c run/build 2> run/build.err")
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(er, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Run error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, tid, cmd, info.ExitCode))
		return
	}

	// Prepare the execdir
	execdir := filepath.Join(w.WorkDir, "execdir")
	if _, err = os.Stat(execdir); os.IsNotExist(err) {
		os.Mkdir(execdir, DirPerm)
	} else {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	// Link the file to execdir
	testcase_in := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.in", tid))
	testcase_out := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.out", tid))
	link_in := filepath.Join(execdir, "testcase.in")
	link_out := filepath.Join(execdir, "testcase.out")
	err = os.Link(testcase_in, link_in)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	err = os.Link(testcase_out, link_out)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}

	// Run testcase
	cmd = "/bin/bash -c run/run execdir/testcase.in execdir/program.out ./program 2> run.err; touch done.lck"
	insp, er := cli.ContainerInspect(ctx, w.containerID)
	if er != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	runinfo, er := w.runProtect(ctx, &insp, insp.State.Pid, uint64(w.JudgeInfo.TimeLimit), w.JudgeInfo.OutputLimit, "execdir/program.out")
	if er != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}

	if runinfo.timeexceed {
		// Perform Post
	}
	if runinfo.memexceed {
		// Perform Post
	}
	if runinfo.outputexceed {
		// Perform Post
	}
	// If not these error, then it is runtime error
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Run error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, tid, cmd, info.ExitCode))
		return
	}
	// Report the run result to server
	return
}
