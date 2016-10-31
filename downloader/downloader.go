package downloader

import (
	"context"
	"github.com/VOID001/D-judge/request"
)

var apiMap = map[string]string{
	"testcase":   "xxx",
	"executable": "xxx",
	"code":       "xxx",
}

type Downloader struct {
	URL         string
	FileType    string
	FileName    string
	MD5         string
	Destination string
	UseCache    bool
	Params      []string
}

func cleanupcache() {

}

func (d *Downloader) Download(ctx context.Context) (err error) {
	return
}

func lookupcache(name string, md5sum string) {

}
