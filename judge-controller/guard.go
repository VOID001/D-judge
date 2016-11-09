package controller

// Internal Run Guard Module

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
)

func (w *Worker) runProtect(ctx context.Context, insp *types.ContainerJSON, pid int, timelim uint64, outputlim int64, outputfile string) (info runinfo, err error) {
	starttime := time.Now().UnixNano()
	curtime := time.Now().UnixNano()

	// Use run protect to protect the running instance
	// It will start right after the execmd =w=
	info.outputexceed = false
	info.usedmem = 0
	info.usedtime = 0
	info.timeexceed = false
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		err = errors.Wrap(err, "run protect error: cannot attach process")
		return
	}
	m, err := p.MemoryInfoEx()
	if err != nil {
		err = errors.Wrap(err, "run protect error: cannot get memory info")
	}
	var f os.FileInfo
	if outputfile != "" {
		f, err = os.Stat(outputfile)
		if err != nil && !os.IsNotExist(err) {
			err = errors.Wrap(err, "run protect error: cannot get output file")
			return
		}
	}
	wt, er := fsnotify.NewWatcher()
	if er != nil {
		err = errors.Wrap(er, "run protect error: cannot create watcher")
		return
	}
	defer wt.Close()
	err = wt.Add(w.WorkDir)
	if err != nil {
		err = errors.Wrap(err, "run protect error: cannot add watchpoint")
		return
	}
	log.Debugf("Add watch to %s", w.WorkDir)
Loop:
	for {
		select {
		case ev := <-wt.Events:
			log.Debugf("%s", ev.String())
			if ev.Op == fsnotify.Create && strings.HasSuffix(ev.Name, "done.lck") {
				curtime = time.Now().UnixNano()
				break Loop
			}
		default:
			curtime = time.Now().UnixNano()
			// Only when output file is not empty, check the file size
			if outputfile != "" {
				f, err = os.Stat(outputfile)
				if err != nil && !os.IsNotExist(err) {
					err = errors.Wrap(err, "run protect error: cannot stat outputfile")
					break Loop
				}
				// Output Limit exceed
				if err == nil && f.Size() > outputlim {
					info.outputexceed = true
					break Loop
				}
			}
			// Collect memory used
			if info.usedmem < m.Dirty {
				info.usedmem = m.Dirty
			}
			// Time limit exceed
			if curtime-starttime > int64(timelim*1000000000) {
				info.timeexceed = true
				log.Debugf("Program exceed hard time limit(used %d, hardlim %d), terminated now", curtime-starttime, timelim*1000000000)
				// Killed the program
				err = p.Terminate()
				if err != nil {
					p.Kill()
				}
				break Loop
			}
			// done.lck create too quick, then just get it and exit
			_, err = os.Stat(filepath.Join(w.WorkDir, "done.lck"))
			if err == nil {
				break Loop
			}

		}
	}
	info.usedtime = curtime - starttime
	info.memexceed = insp.State.OOMKilled
	err = os.RemoveAll(filepath.Join(w.WorkDir, "done.lck"))
	if err != nil {
		err = errors.Wrap(err, "cannot remove done.lck [ABORT!]")
		return
	}
	return
}

func (w *Worker) execcmd(ctx context.Context, cli *client.Client, user string, cmd string) (info types.ContainerExecInspect, err error) {
	ec := types.ExecConfig{}
	ec.Detach = true
	ec.Tty = false
	ec.AttachStdout = true // Set this to true will make the command block until finish
	ec.Cmd = make([]string, 3)
	ec.Cmd[0] = "/bin/bash"
	ec.Cmd[1] = "-c"
	ec.Cmd[2] = cmd
	log.Infof("%+v", ec)
	ec.User = user
	eresp, er := cli.ContainerExecCreate(ctx, w.containerID, ec)
	if er != nil {
		err = errors.Wrap(er, "exec command in container error")
		return
	}
	sc := types.ExecStartCheck{}
	sc.Tty = ec.Tty
	sc.Detach = ec.Detach
	log.Infof("ExecStartCheck %+v", sc)
	err = cli.ContainerExecStart(ctx, eresp.ID, sc)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
		return
	}
	log.Debugf("Executing exec ID = %s", eresp.ID)
	info, err = cli.ContainerExecInspect(ctx, eresp.ID)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
		return
	}
	log.Infof("ONLY FOR CURRENT DEBUG info.ExitCode = %d", info.ExitCode)
	return
}

func (w *Worker) execcmdAttach(ctx context.Context, cli *client.Client, user string, cmd string) (info types.ContainerExecInspect, err error) {
	ec := types.ExecConfig{}
	ec.Detach = false
	ec.Tty = false
	ec.AttachStdout = true // Set this to true will make the command block until finish
	ec.Cmd = make([]string, 3)
	ec.Cmd[0] = "/bin/bash"
	ec.Cmd[1] = "-c"
	ec.Cmd[2] = cmd
	log.Infof("%+v", ec)
	ec.User = user
	eresp, er := cli.ContainerExecCreate(ctx, w.containerID, ec)
	if er != nil {
		err = errors.Wrap(er, "exec command in container error")
		return
	}
	sc := types.ExecStartCheck{}
	sc.Tty = ec.Tty
	sc.Detach = ec.Detach
	log.Infof("ExecStartCheck %+v", sc)
	err = cli.ContainerExecStart(ctx, eresp.ID, sc)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
		return
	}
	insp, err := cli.ContainerExecAttach(ctx, eresp.ID, ec)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
	}
	c := insp.Conn
	defer insp.Close()

	log.Debugf("Executing exec ID = %s", eresp.ID)
	one := make([]byte, 1)
	_, err = c.Read(one)
	if err == io.EOF {
		log.Infof("Connection Closed")
	}
	info, err = cli.ContainerExecInspect(ctx, eresp.ID)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
		return
	}
	return
}
