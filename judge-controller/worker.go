package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/downloader"
)

type Worker struct {
	JudgeInfo    config.JudgeInfo
	WorkDir      string
	DockerImage  string
	RunUser      string
	MaxRetryTime int
}

const (
	FilePerm = 0644
	DirPerm  = 0755
)

func (w *Worker) prepare(ctx context.Context) (err error) {
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

	// Get the build & run script then
	d = downloader.Downloader{
		FileType:     "executable",
		FileName:     w.JudgeInfo.RunZip,
		Destination:  filepath.Join(w.WorkDir, w.JudgeInfo.RunZip),
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

	d.FileName = w.JudgeInfo.BuildZip
	d.Destination = filepath.Join(w.WorkDir, w.JudgeInfo.BuildZip)
	d.MD5 = w.JudgeInfo.BuildZipMD5
	d.Params = []string{w.JudgeInfo.BuildZip}
	err = d.Do(ctx)

	d.FileName = w.JudgeInfo.CompareZip
	d.Destination = filepath.Join(w.WorkDir, w.JudgeInfo.CompareZip)
	d.MD5 = w.JudgeInfo.CompareZipMD5
	d.Params = []string{w.JudgeInfo.CompareZip}

	err = d.Do(ctx)
	if err != nil {
		err = errors.Wrap(err, "error preparing for judge")
		return
	}

	return
}

func (w *Worker) build() {

}

func (w *Worker) run() {

}

func (w *Worker) judge() {

}

func (w *Worker) cleanup() {

}
