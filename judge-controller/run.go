package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/request"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
)

func (w *Worker) run(ctx context.Context, rank int64, tid int64) (ok bool, err error) {
	// Prepare the run script
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, config.GlobalConfig.DockerVersion, nil, nil)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}

	insp, er := cli.ContainerInspect(ctx, w.containerID)

	// Prepare the execdir
	execdir := filepath.Join(w.WorkDir, "execdir")
	if _, err = os.Stat(execdir); os.IsNotExist(err) {
		os.Mkdir(execdir, DirPerm)
	} else {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}

	// Link the file to execdir
	testcase_in := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.in", rank))
	testcase_out := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.out", rank))
	link_in := filepath.Join(execdir, "testcase.in")
	link_out := filepath.Join(execdir, "testcase.out")
	err = os.Link(testcase_in, link_in)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	err = os.Link(testcase_out, link_out)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}

	// Run testcase
	pid := insp.State.Pid
	//cmd = "/bin/bash -c run/run execdir/testcase.in execdir/program.out ./program 2> run.err; touch ./done.lck"
	cmd := "run/run execdir/testcase.in execdir/program.out ./program 2> run.err; touch ./done.lck"
	info, err := w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	runinfo, er := w.runProtect(ctx, &insp, pid, uint64(w.JudgeInfo.TimeLimit), w.JudgeInfo.OutputLimit, "execdir/program.out")
	log.Debugf("run protect protecting %s", cmd)
	log.Infof("run protect [run] done, runinfo %+v", runinfo)
	if er != nil {
		err = errors.Wrap(err, fmt.Sprintf("Run error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	log.Debugf("Testcase run done, info %+v", runinfo)

	// Report the result if run error
	res := config.RunResult{}
	res.RunTime = float64(runinfo.usedtime) * 1.0 / 1000 / 1000 / 1000
	res.TestcaseID = tid
	res.JudgingID = w.JudgeInfo.JudgingID

	// Parse system meta and send to output system
	// For Domjudge compability

	// Make the systemMeta look like this = =
	/*
		Timelimit exceeded.
		runtime: 1.860s cpu, 2.200s wall
		memory used: 131072 bytes
	*/
	res.OutputSystem = fmt.Sprintf("%s.\nruntime: %fs cpu, %fs wall:\nmemory used: %dbytes\n", res.RunResult, res.RunTime, res.RunTime, runinfo.usedmem)
	log.Debugf("system meta %s", res.OutputSystem)
	// Save for Judge use
	ioutil.WriteFile(filepath.Join(execdir, "program.meta"), []byte(res.OutputSystem), FilePerm)

	res.RunResult = ""
	if runinfo.timeexceed {
		res.RunResult = config.ResTLE
	}
	if runinfo.memexceed {
		res.RunResult = config.ResRE
	}
	if runinfo.outputexceed {
		res.RunResult = config.ResRE
	}
	// If not these error, then it is runtime error
	if info.ExitCode != 0 {
		res.RunResult = config.ResRE
		reinfo, er := ioutil.ReadFile(filepath.Join(w.WorkDir, "run.err"))
		if er != nil {
			err = errors.Wrap(er, "run error")
			return
		}
		res.OutputError = fmt.Sprintf("%s", reinfo)
	}

	// Run error, post to Server
	if res.RunResult != "" {
		err = request.PostResult(ctx, res)
		if err != nil {
			err = errors.Wrap(err, "run error")
			return
		}
		ok = false
		return
	}
	ok = true
	return
}
