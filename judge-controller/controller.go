package controller

import (
	"context"
	"fmt"
	"path/filepath"

	"runtime"
	"sync"

	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/downloader"
	"github.com/VOID001/D-judge/request"

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
	cli, err := client.NewClient(config.GlobalConfig.DockerServer, config.GlobalConfig.DockerVersion, nil, nil)
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
		// Only Judge Error Will Processed here, other error will process
		// in the worker function

		if w, ok := <-d.workerChan; ok {
			log.Infof("Started Judging RunID #%d, running on CPU %d", w.JudgeInfo.SubmitID, cpuid)
			w.CPUID = cpuid
			err := w.prepare(ctx)
			if err != nil {
				log.Error(err)
				request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
				continue // Future will change to continue
			}
			log.Infof("RunID #%d prepare OK", w.JudgeInfo.SubmitID)
			ok, err := w.build(ctx)
			if err != nil {
				w.cleanup(ctx)
				log.Error(err)
				request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
				continue
			}
			// Compile Error, stop the current test
			if !ok {
				w.cleanup(ctx)
				continue
			}
			log.Infof("RunID #%d compile OK", w.JudgeInfo.SubmitID)
			for {
				// Request for testcase
				tinfo := config.TestcaseInfo{}

				err = request.Do(ctx, http.MethodGet, fmt.Sprintf("/testcases?judgingid=%d", w.JudgeInfo.SubmitID), nil, "", &tinfo)
				if err != nil {
					request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
					break
					// Return Judge Error
				}
				if tinfo.TestcaseID == 0 {
					break
				}
				log.Debugf("Testcase info %+v", tinfo)

				dl := downloader.Downloader{}
				dl.FileType = "testcase"
				dl.Destination = filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.in", tinfo.Rank))
				dl.FileName = fmt.Sprintf("%d-%s.in", tinfo.TestcaseID, tinfo.MD5SumInput)
				dl.SkipMD5Check = false
				dl.MD5 = tinfo.MD5SumInput
				dl.UseCache = true
				dl.Params = []string{fmt.Sprintf("%d", tinfo.TestcaseID), "input"}
				err = dl.Do(ctx)
				if err != nil {
					err = errors.Wrap(err, "worker error: downloading testcase error")
					log.Error(err)
					request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
					break
					// Return Judge Error
				}

				dl.Destination = filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d.out", tinfo.Rank))
				dl.FileName = fmt.Sprintf("%d-%s.out", tinfo.TestcaseID, tinfo.MD5SumInput)
				dl.MD5 = tinfo.MD5SumOutput
				dl.Params = []string{fmt.Sprintf("%d", tinfo.TestcaseID), "output"}
				err = dl.Do(ctx)
				if err != nil {
					err = errors.Wrap(err, "worker error: downloading testcase error")
					log.Error(err)
					request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
					break
				}

				// Run testcase
				ok, err = w.run(ctx, tinfo.Rank, tinfo.TestcaseID)
				if err != nil {
					w.cleanup(ctx)
					err = errors.Wrap(err, "worker error")
					log.Error(err)
					request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
					break
				}
				if !ok {
					break
				}
				log.Infof("Run Testcase %d OK", tinfo.Rank)

				// Judge testcase
				err = w.judge(ctx, tinfo.Rank, tinfo.TestcaseID)
				if err != nil {
					w.cleanup(ctx)
					err = errors.Wrap(err, "worker error")
					log.Error(err)
					request.JudgeError(ctx, err, w.JudgeInfo.JudgingID)
					break
				}
				log.Infof("Judge Testcase %d OK", tinfo.Rank)

			}
			err = w.cleanup(ctx)
			if err != nil {
				log.Error(err)
				continue
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
