package controller

import (
	"context"

	"github.com/VOID001/D-judge/config"
	//"github.com/VOID001/D-judge/request"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"runtime"
	"sync"
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

func (d *Daemon) Run(ctx context.Context, w *Worker) (err error) {
	if d.CurrentWorker >= d.MaxWorker {
		err = errors.New(ErrMaxWorkerExceed)
		return
	}
	mu.Lock()
	d.CurrentWorker++
	mu.Unlock()

	w.prepare(ctx)

	w.build()

	w.run()

	w.judge()

	mu.Lock()
	d.CurrentWorker--
	mu.Unlock()
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
	return
}
