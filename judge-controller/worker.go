package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/downloader"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/shirou/gopsutil/process"
)

type runinfo struct {
	usedmem      uint64
	usedtime     uint64
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

func (w *Worker) prepare(ctx context.Context) (err error) {
	log.Debugf("preparing for judge, worker info %+v", w)
	// Download needed sources, perpare the working dir

	// Ensure the robustness of the judgehost
	if _, err = os.Stat(w.WorkDir); os.IsNotExist(err) {
		log.Errorf("work dir %s not found, re-create work dir", w.WorkDir)
		os.MkdirAll(w.WorkDir, DirPerm)
	}

	// Get the code first
	d := downloader.Downloader{
		FileType:     "code",
		Destination:  filepath.Join(w.WorkDir, "foo"), // Here just provide a dummy destination, it will correct when call downloader
		FileName:     filepath.Join(w.WorkDir, "foo"), // Here just provide a dummy filename, it will correct when call downloader
		SkipMD5Check: true,
		UseCache:     false,
		Params:       []string{fmt.Sprintf("%d", w.JudgeInfo.SubmitID)},
	}
	err = d.Do(ctx)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}
	w.codeFileName = d.FileName

	// Get the build & run script then
	rundir := filepath.Join(w.WorkDir, "run")
	err = os.Mkdir(rundir, DirPerm)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	d = downloader.Downloader{
		FileType:     "executable",
		FileName:     w.JudgeInfo.RunZip,
		Destination:  filepath.Join(rundir, w.JudgeInfo.RunZip),
		SkipMD5Check: false,
		MD5:          w.JudgeInfo.RunZipMD5,
		UseCache:     true,
		Params:       []string{w.JudgeInfo.RunZip},
	}
	err = d.Do(ctx)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	builddir := filepath.Join(w.WorkDir, "build")
	err = os.Mkdir(builddir, DirPerm)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	d.FileName = w.JudgeInfo.BuildZip
	d.Destination = filepath.Join(builddir, w.JudgeInfo.BuildZip)
	d.MD5 = w.JudgeInfo.BuildZipMD5
	d.Params = []string{w.JudgeInfo.BuildZip}
	err = d.Do(ctx)

	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	comparedir := filepath.Join(w.WorkDir, "compare")
	err = os.Mkdir(comparedir, DirPerm)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}
	d.FileName = w.JudgeInfo.CompareZip
	d.Destination = filepath.Join(comparedir, w.JudgeInfo.CompareZip)
	d.MD5 = w.JudgeInfo.CompareZipMD5
	d.Params = []string{w.JudgeInfo.CompareZip}

	err = d.Do(ctx)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	return
}

func (w *Worker) build(ctx context.Context) (err error) {
	// Start the container and Build the target
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, "build error")
		return
	}
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
		err = errors.Wrap(er, "build error")
		return
	}
	defer cli.ContainerRemove(ctx, w.containerID, types.ContainerRemoveOptions{})
	w.containerID = resp.ID
	log.Debugf("RunID #%d container create ID %s", w.JudgeInfo.SubmitID, w.containerID)
	err = cli.ContainerStart(ctx, w.containerID, types.ContainerStartOptions{})
	if err != nil {
		err = errors.Wrap(err, "build error")
		return
	}

	cmd := fmt.Sprintf("bash -c unzip -o build/%s -d build", w.JudgeInfo.BuildZip)
	log.Infof("container %s executing %s", w.containerID, cmd)
	info, err := w.execcmd(ctx, cli, "root", cmd)
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("build error: RunID#%d exec command %+v return non-zero value %d", w.JudgeInfo.SubmitID, cmd, info.ExitCode))
		return
	}

	cmd = "bash -c build/build 2> build.err"
	log.Infof("container %s executing %s", w.containerID, cmd)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("build error: exec command %+v return non-zero value %d", cmd, info.ExitCode))
		return
	}
	// Do the real compile
	insp, err := cli.ContainerInspect(ctx, w.containerID)
	if err != nil {
		err = errors.Wrap(err, "build error: inspect container")
		return
	}
	pid := insp.State.Pid
	cmd = fmt.Sprintf("bash -c build/run ./program DUMMY ./%s 2> ./compile.err > ./compile.out; touch ./done.lck", w.codeFileName)
	//cmd = fmt.Sprintf("./run ../program DUMMY ../%s", w.codeFileName)
	log.Debugf("container %s executing %s", w.containerID, cmd)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	runinfo, er := w.runProtect(ctx, &insp, pid, uint64(30), w.JudgeInfo.OutputLimit, "")
	if er != nil {
		err = errors.Wrap(er, "build error")
	}
	log.Infof("run protect exited, runinfo %+v", runinfo)
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("build error: exec command %+v return non-zero value %d", cmd, info.ExitCode))
		return
	}
	return
}

func (w *Worker) runProtect(ctx context.Context, insp *types.ContainerJSON, pid int, timelim uint64, outputlim int64, outputfile string) (info runinfo, err error) {
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

	for {
		_, er := os.Stat(filepath.Join(w.WorkDir, "done.lck"))
		if er == nil {
			err = os.Remove(filepath.Join(w.WorkDir, "done.lck")) // Suppose it will never fail
			info.memexceed = insp.State.OOMKilled
			return
		}
		datmem := m.Dirty // This should be Data Field, but Now Data Field value is wrong
		if datmem > info.usedmem {
			info.usedmem = datmem
		}
		if outputfile != "" {
			sz := f.Size()
			if sz > outputlim {
				info.outputexceed = true
				return
			}
		}
		info.usedtime++
		if info.usedtime > timelim*1000 {
			info.timeexceed = true
			p.Kill()
			return
		}
		time.Sleep(time.Millisecond)
	}
	return
}

func (w *Worker) execcmd(ctx context.Context, cli *client.Client, user string, cmd string) (info types.ContainerExecInspect, err error) {
	ec := types.ExecConfig{}
	ec.Detach = true
	ec.Cmd = strings.Split(cmd, " ")
	excmd := strings.Join(ec.Cmd[2:], " ")
	ec.Cmd[2] = excmd
	ec.User = user
	eresp, er := cli.ContainerExecCreate(ctx, w.containerID, ec)
	if er != nil {
		err = errors.Wrap(er, "exec command in container error")
		return
	}
	sc := types.ExecStartCheck{}
	log.Infof("exec ID = %s", eresp.ID)
	err = cli.ContainerExecStart(ctx, eresp.ID, sc)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
		return
	}
	//insp, err := cli.ContainerExecAttach(ctx, eresp.ID, ec)
	//if err != nil {
	//	err = errors.Wrap(err, "exec command in container error")
	//}
	//defer insp.Close()
	log.Infof("Executing exec ID = %s", eresp.ID)
	info, err = cli.ContainerExecInspect(ctx, eresp.ID)
	if err != nil {
		err = errors.Wrap(err, "exec command in container error")
	}
	//buf := bytes.Buffer{}
	//buf.ReadFrom(insp.Reader)
	return
}

func (w *Worker) run(ctx context.Context) {
	// Build the run script

}

func (w *Worker) judge(ctx context.Context) {
	// Build the judge script
}

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
