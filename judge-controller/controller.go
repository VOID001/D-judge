package controller

import (
	"context"

	"github.com/VOID001/D-judge/config"
	//"github.com/VOID001/D-judge/request"
	"runtime"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	//"github.com/docker/engine-api/types"
	//"github.com/docker/engine-api/types/container"
	"net/http"
)

var mu sync.Mutex
var cpuMap []bool
var cpuCount int

const (
	ErrMaxWorkerExceed = "max worker exceed"
)

type Daemon struct {
	MaxWorker     int
	CurrentWorker int
	WorkerState   []string
	workerChan    chan Worker
	resultChan    chan RunResult
}

type RunResult struct {
	Stage string
	RunID int
	err   error
}

var httpcli http.Client

func init() {
	cpuCount = runtime.NumCPU()
	cpuMap = make([]bool, cpuCount)
	httpcli = http.Client{}
}

func Ping(ctx context.Context) (err error) {
	cli, err := client.NewClient(config.GlobalConfig.DockerServer, "v1.24", nil, nil)
	if err != nil {
		err = errors.Wrap(err, "create docker client error")
		return err
	}
	_, err = cli.Info(ctx)
	if err != nil {
		err = errors.Wrap(err, "ping docker server error")
		return err
	}
	return
}

func (d *Daemon) AddTask(ctx context.Context, jinfo config.JudgeInfo, dir string, img string) (err error) {
	log.Debugf("call AddTask(context, jinfo = %+v, dir = %+v, img = %+v)", jinfo, dir, img)
	w := Worker{}
	w.JudgeInfo = jinfo
	w.WorkDir = dir
	w.RunUser = "root"
	w.DockerImage = img
	d.workerChan <- w
	return
}

func (d *Daemon) Run(ctx context.Context) {
	d.workerChan = make(chan Worker, 100)
	for i := 0; i < d.MaxWorker; i++ {
		go d.run(ctx, i)
	}
	return
}

func (d *Daemon) run(ctx context.Context, cpuid int) {
	for {
		if w, ok := <-d.workerChan; ok {
			log.Infof("Started Judging RunID #%d, running on CPU %d", w.JudgeInfo.SubmitID, cpuid)
			w.CPUID = cpuid
			err := w.prepare(ctx)
			if err != nil {
				w.cleanup(ctx)
				log.Error(err)
				return
			}
			err = w.build(ctx)
			if err != nil {
				w.cleanup(ctx)
				log.Error(err)
				return
			}
			err = w.run(ctx)
			err = w.judge(ctx)
			err = w.cleanup(ctx)
			if err != nil {
				log.Error(err)
				return
			}
		} else {
			break
		}
	}
	return
}

func GetAvailableCPU(ctx context.Context) (cpuid int, err error) {
	for i := 0; i < cpuCount; i++ {
		if cpuMap[i] != true {
			mu.Lock()
			cpuMap[i] = true
			cpuid = i
			defer mu.Unlock()
			return
		}
	}
	cpuid = -1
	return
}
