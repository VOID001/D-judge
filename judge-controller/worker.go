package controller

import (
	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/downloader"
)

type Worker struct {
	JudgeInfo    config.JudgeInfo
	DockerImage  string
	RunUser      string
	MaxRetryTime int
}

func (w *Worker) prepare() {

}

func (w *Worker) build() {

}

func (w *Worker) run() {

}

func (w *Worker) judge() {

}
