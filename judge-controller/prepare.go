package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/downloader"
	"github.com/pkg/errors"
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
