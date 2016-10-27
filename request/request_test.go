package request

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/pkg/errors"
)

var GlobalConfig = config.SystemConfig{
	HostName:         "dev-test",
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
	// Set Level to debug
	log.SetLevel(log.DebugLevel)
}

func TestDoPostForm(t *testing.T) {
	t.Logf("Running TestDo")
	body := url.Values{"hostname": {config.GlobalConfig.HostName}}
	err := Do(context.Background(), http.MethodPost, "/judgehosts", body, TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "post form error")
		t.Error(err)
		t.Fail()
	}
}

func TestDoPostJSON(t *testing.T) {

}

func TestGet(t *testing.T) {

}

func TestDoPostJudgings(t *testing.T) {
	jinfo := config.JudgeInfo{}
	err := Do(context.Background(), http.MethodPost, fmt.Sprintf("/judgings?judgehost=%s", config.GlobalConfig.HostName), nil, "", &jinfo)
	if err != nil {
		err = errors.Wrap(err, "post judgings error")
		t.Error(err)
		t.Fail()
	}
	t.Logf("Judge Info Get\n %+v", jinfo)
	return
}
