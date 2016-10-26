package controller

import (
	"context"
	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	//"github.com/docker/engine-api/types"
	//"github.com/docker/engine-api/types/container"
	"net/http"
)

type Daemon struct {
}

var httpcli http.Client

func init() {
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
