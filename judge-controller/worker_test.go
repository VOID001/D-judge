package controller

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"path/filepath"
	"testing"
)

var GlobalConfig = config.SystemConfig{
	HostName:         "judge-01",
	EndpointName:     "neuoj",
	EndpointUser:     "neuoj",
	EndpointURL:      "http://localhost:8080/api",
	EndpointPassword: "neuoj",
	JudgeRoot:        "/tmp/judge_root",
	CacheRoot:        "/tmp/cache_root",
	DockerImage:      "void001/neuoj-judge-image:latest",
	DockerServer:     "unix:///var/run/docker.sock",
}

func init() {
	config.GlobalConfig = GlobalConfig
	log.SetLevel(log.DebugLevel)
}

func TestWorkerPrepare(t *testing.T) {
	w := Worker{}
	w.JudgeInfo = config.JudgeInfo{
		SubmitID:      1,
		ContestID:     0,
		TeamID:        2,
		JudgingID:     1,
		ProblemID:     1,
		Language:      "c",
		TimeLimit:     3,
		MemLimit:      104,
		OutputLimit:   0,
		BuildZip:      "c",
		BuildZipMD5:   "c76e6afa913a9fc827c42c2357f47a53",
		RunZip:        "run",
		RunZipMD5:     "c2cb7864f2f7343d1ab5094b8fd40da4",
		CompareZip:    "compare",
		CompareZipMD5: "71306aae6e243f8a030ab1bd7d6b354b",
		CompareArgs:   "",
	}
	w.WorkDir = filepath.Join(config.GlobalConfig.JudgeRoot, "judge-test-1")
	err := w.prepare(context.Background())
	if err != nil {
		t.Logf("Failed, error: %+v", err)
		t.Fail()
		return
	}
}
