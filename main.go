package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/judge-controller"
	"github.com/VOID001/D-judge/request"

	"github.com/pkg/errors"
)

// Log level constant
const (
	INFO    = 0    // INFO Level
	WARN    = 1    // WARN Level
	DEBUG   = 2    // DEBUG Level
	DirPerm = 0744 // Dir Permission
)

// Error constants
const (
	ErrNoDir  = "no such file or directory"
	ErrNoFile = "no such file or directory"
)

var path string
var debuglv int64

// GlobalConfig Config Object contain the global system config
var GlobalConfig config.SystemConfig

func init() {
	flag.StringVar(&path, "c", "config.toml", "select configuration file")
	flag.Int64Var(&debuglv, "d", 0, "debug mode enabled")
	flag.Parse()
	_, err := toml.DecodeFile(path, &config.GlobalConfig)
	GlobalConfig = config.GlobalConfig
	if err != nil {
		err = errors.Wrap(err, "Processing config file error")
		log.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.Wrap(err, "Get current directory error")
	}
	if GlobalConfig.JudgeRoot[0] != '/' {
		GlobalConfig.JudgeRoot = filepath.Join(cwd, GlobalConfig.JudgeRoot)
	}
	if GlobalConfig.CacheRoot[0] != '/' {
		GlobalConfig.CacheRoot = filepath.Join(cwd, GlobalConfig.CacheRoot)
	}
	if debuglv == INFO {
		log.SetLevel(log.InfoLevel)
	}
	if debuglv == WARN {
		log.SetLevel(log.WarnLevel)
	}
	if debuglv == DEBUG {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	log.Debugf("Settings %+v", GlobalConfig)
	// Perform Sanity Check
	log.Infof("sanity check start")
	err := sanityCheckDir(GlobalConfig.JudgeRoot)
	if err != nil {
		err = errors.Wrap(err, "sanity check dir judgeroot error")
		log.Fatal(err)
	}
	err = sanityCheckDir(GlobalConfig.CacheRoot)
	if err != nil {
		err = errors.Wrap(err, "sanity check dir cacheroot error")
		log.Fatal(err)
	}
	err = sanityCheckConnection(GlobalConfig.EndpointURL)
	if err != nil {
		err = errors.Wrap(err, "sanity check connection error")
		log.Fatal(err)
	}
	err = sanityCheckDocker()
	if err != nil {
		err = errors.Wrap(err, "sanity check docker error")
		log.Fatal(err)
	}

	err = request.Do(context.Background(), http.MethodPost, "/judgehosts", url.Values{"hostname": {config.GlobalConfig.HostName}}, request.TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "main loop error")
		log.Fatal(err)
	}
	log.Infof("sanity check success")

	// PerformRequest Lifcycle
	for {
		jinfo := config.JudgeInfo{}
		// Error When Requesting Judgehost
		if err != nil {
			log.Warn(err)
		}
		// Request For Judge
		err = request.Do(context.Background(), http.MethodPost, fmt.Sprintf("/judgings?judgehost=%s", config.GlobalConfig.HostName), nil, "", &jinfo)
		if err != nil {
			log.Warn(err)
		}
		if jinfo.SubmitID != 0 {
			workDir := fmt.Sprintf("%s/c%d-s%d-j%d", config.GlobalConfig.JudgeRoot, jinfo.ContestID, jinfo.SubmitID, jinfo.JudgingID)
			if _, err := os.Stat(workDir); err == nil {
				oldWorkDir := fmt.Sprintf("%s-old-%d", workDir, time.Now().Unix())
				err := os.Rename(workDir, oldWorkDir)
				if err != nil {
					err = errors.Wrap(err, "main loop error")
					log.Fatal(err)
				}
			}
			os.Mkdir(workDir, DirPerm)
		}
	}

	// Start Compile

	// Start Run

	// Start Compare

	// NextTestcase
}

func sanityCheckDir(dir string) (err error) {
	_, err = ioutil.ReadDir(dir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(dir, DirPerm)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("cannot make %s", dir))
			return
		}
		log.Infof("created dir %s with mode %04o", dir, DirPerm)
	}
	info, err := os.Stat(dir)
	log.Infof("dir %s mode bits %s", dir, info.Mode())
	return
}

func sanityCheckConnection(endpoint string) (err error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	req.SetBasicAuth(config.GlobalConfig.EndpointUser, config.GlobalConfig.EndpointPassword)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("cannot create request %s", endpoint))
		return
	}
	cli := http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("cannot connect to %s", endpoint))
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Errorf("body close error %s", err.Error())
		}
	}()
	return
}

func sanityCheckDocker() (err error) {
	err = controller.Ping(context.Background())
	if err != nil {
		err = errors.Wrap(err, "docker Ping error")
	}
	return
}
