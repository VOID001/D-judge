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

const (
	ExitAC = 42
	ExitWA = 43
)

func (w *Worker) judge(ctx context.Context, tid int64) (err error) {
	// Create testcase dir, use to store result
	execdir := filepath.Join(w.WorkDir, "execdir")
	testcasedir := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d", tid))
	err = os.Mkdir(testcasedir, DirPerm)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	// Build the judge script
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	cmd := fmt.Sprintf("/bin/bash -c unzip -o compare/%s -d compare", w.JudgeInfo.CompareZip)
	info, er := w.execcmd(ctx, cli, "root", cmd)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Judge error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, tid, cmd, info.ExitCode))
	}

	cmd = fmt.Sprintf("/bin/bash -c compare/build 2> compare/build.err", w.JudgeInfo.CompareZip)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Judge error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, tid, cmd, info.ExitCode))
	}

	cmd = fmt.Sprintf("/bin/bash compare/run execdir/testcase.in execdir/testcase.out testcase001 < execdir/program.out 2> compare.err >compare.out")
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	switch info.ExitCode {
	case ExitWA:
		// Report Accepted
	case ExitAC:
		// Report Wrong Answer
	default:
		// Report Judge Error
	}

	// Remove execdir for next time use
	err = os.RemoveAll(execdir)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, tid))
		return
	}
	return
}
