package controller

import (
	"context"
	"path/filepath"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
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
	DockerVersion:    "v1.24",
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

func TestWorkerExecCMD(t *testing.T) {
	w := Worker{}
	//cmd := fmt.Sprintf("compare/run execdir/testcase.in execdir/testcase.out testcase001 < execdir/program.out 2> compare.err >compare.out")
	cmd := "sleep 5; exit 233"
	cli, err := client.NewClient(config.GlobalConfig.DockerServer, config.GlobalConfig.DockerVersion, nil, nil)
	if err != nil {
		t.Logf("Failed error: %+v", err)
		t.Fail()
		return
	}

	cfg := container.Config{}
	cfg.Image = config.GlobalConfig.DockerImage
	cfg.User = "root" // Future will change to judge, a low-privileged user
	cfg.Tty = true
	cfg.WorkingDir = "/sandbox"
	cfg.AttachStdin = false
	cfg.AttachStderr = false
	cfg.AttachStdout = true
	cfg.Cmd = []string{"/bin/bash"}
	hcfg := container.HostConfig{}
	hcfg.Binds = []string{"/tmp/testdir:/sandbox"}
	hcfg.Memory = config.GlobalConfig.RootMemory
	hcfg.PidsLimit = 64 // This is enough for almost all case

	resp, err := cli.ContainerCreate(context.TODO(), &cfg, &hcfg, nil, "")
	if err != nil {
		t.Logf("Failed error: %+v", err)
		t.Fail()
		return
	}
	err = cli.ContainerStart(context.TODO(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Logf("Failed error: %+v", err)
		t.Fail()
		return
	}
	w.containerID = resp.ID
	info, err := w.execcmd(context.TODO(), cli, "root", cmd)
	if err != nil {
		t.Logf("Failed error: %+v", err)
		t.Fail()
		return
	}
	if info.ExitCode != 233 {
		t.Logf("Expected exit code 233, got %d", info.ExitCode)
		t.Fail()
		return
	}

}
