package downloader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
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

func TestDoWithoutCache(t *testing.T) {
	d := Downloader{}
	d.Destination = "/tmp/testdata"
	d.FileName = "testdata"
	d.SkipMD5Check = true
	d.UseCache = false
	d.FileType = "code"
	d.Params = []string{"1"}
	err := d.Do(context.Background())
	if err != nil {
		t.Logf("downloader do error: %+v", err)
		t.Fail()
		return
	}

	if _, err := os.Stat(d.Destination); err != nil && os.IsNotExist(err) {
		t.Logf("download failed but downloader do not return error")
		t.Fail()
		return
	}
	return
}

func TestDoWithCache(t *testing.T) {
	d := Downloader{}
	d.Destination = "/tmp/testdata"
	d.FileName = "testdata"
	d.SkipMD5Check = false
	d.UseCache = true
	d.FileType = "code"
	d.Params = []string{"1"}
	d.MD5 = "f7af11c0363fafa66f1705058f1a0058"
	err := d.Do(context.Background())
	if err != nil {
		t.Logf("downloader do error: %+v", err)
		t.Fail()
		return
	}
	if _, err := os.Stat(d.Destination); err != nil && os.IsNotExist(err) {
		t.Logf("download failed but downloader do not return error")
		t.Fail()
		return
	}
	if _, err := os.Stat(filepath.Join(config.GlobalConfig.CacheRoot, d.FileName)); err != nil && os.IsNotExist(err) {
		t.Logf("download failed but downloader do not return error")
		t.Fail()
		return
	}
	return
}

func TestDownExecutableWithCache(t *testing.T) {
	d := Downloader{}
	d.Destination = "/tmp/c.zip"
	d.FileName = "c.zip"
	d.SkipMD5Check = false
	d.UseCache = true
	d.FileType = "executable"
	d.Params = []string{"c"}
	d.MD5 = "c76e6afa913a9fc827c42c2357f47a53"
	err := d.Do(context.Background())
	if err != nil {
		t.Logf("downloader do error: %+v", err)
		t.Fail()
		return
	}

	if _, err := os.Stat("/tmp/cache_root/testdata"); err != nil && os.IsNotExist(err) {
		t.Logf("download failed but downloader do not return error")
		t.Fail()
		return
	}
	return
}
