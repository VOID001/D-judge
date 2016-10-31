package downloader

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/VOID001/D-judge/config"
	"github.com/VOID001/D-judge/request"
	"github.com/pkg/errors"
)

var apiMap = map[string]string{
	"testcase":   "/testcase_files?testcaseid=%s&%%s", // Seems strange but works!
	"executable": "/executable?execid=%s",
	"code":       "/submission_files?id=%s",
}

type Downloader struct {
	FileType     string
	FileName     string
	MD5          string
	Destination  string
	UseCache     bool
	Params       []string
	SkipMD5Check bool
}

const (
	DirPerm       = 0755
	FilePerm      = 0644
	CacheContent  = "content"
	CacheChecksum = "checksum"
)

func cleanupcache() (err error) {
	err = os.RemoveAll(config.GlobalConfig.CacheRoot)
	if err != nil {
		err = errors.Wrap(err, "error clean up cache")
		return
	}
	err = os.Mkdir(config.GlobalConfig.CacheRoot, DirPerm)
	if err != nil {
		err = errors.Wrap(err, "error clean up cache")
		return
	}
	return
}

func (d *Downloader) Do(ctx context.Context) (err error) {
	var content string
	url := apiMap[d.FileType]
	for i := 0; i < len(d.Params); i++ {
		url = fmt.Sprintf(url, d.Params[i])
		log.Debugf("url = %s", url)
	}

	hit := d.UseCache
	if d.UseCache {
		// All errors when lookup cache is not fatal, just fallback to no cache mode
		path, er := lookupcache(d.FileName, d.MD5)
		if er != nil {
			err = errors.Wrap(er, fmt.Sprintf("error processing download, downloader info %+v", d))
			log.Error(err)
			log.Infof("Fall back to no cache mode")
			hit = false
		}

		// Cached data found, return now
		if hit && path != "" {
			os.Link(path, d.Destination)
			return
		}
	}

	switch d.FileType {
	case "code":
		// Provide the code name
		m := []map[string]string{}
		err = request.Do(ctx, http.MethodGet, url, nil, "", &m)
		if err != nil {
			err = errors.Wrap(err, "error processing download")
			return
		}
		content = m[0]["content"]
		d.Destination = filepath.Join(filepath.Dir(d.Destination), m[0]["filename"])
		d.FileName = m[0]["filename"]
		break
	default:
		err = request.Do(ctx, http.MethodGet, url, nil, "", &content)
		if err != nil {
			err = errors.Wrap(err, "error processing download")
			return
		}
		break
	}

	// Decode the base64 data
	data, er := base64.StdEncoding.DecodeString(content)
	if er != nil {
		er = errors.Wrap(er, "error processing download")
	}
	// Check MD5
	if !d.SkipMD5Check {
		checksum := md5.Sum(data)
		log.Debugf("checksum = %x, d.MD5 = %s", checksum, d.MD5)
		if fmt.Sprintf("%x", checksum) != d.MD5 {
			err = errors.New("error processing download: checksum error, file corrupted during download")
			return
		}
	}

	if d.SkipMD5Check {
		log.Debugf("MD5 checksum skipped")
	}

	err = ioutil.WriteFile(d.Destination, data, FilePerm)
	if err != nil {
		err = errors.Wrap(err, "error processing download")
	}

	// Save cache errors is not fatal
	if d.UseCache && !hit {
		log.Debugf("Cache not hit")
		os.Mkdir(filepath.Join(config.GlobalConfig.CacheRoot, d.FileName), DirPerm)
		cachedata := filepath.Join(config.GlobalConfig.CacheRoot, d.FileName, CacheContent)
		err = os.Link(d.Destination, cachedata)
		if err != nil {
			log.Errorf("save into cache failed, error %+v", err)
		}
		cachemd5 := filepath.Join(config.GlobalConfig.CacheRoot, d.FileName, CacheChecksum)
		err = ioutil.WriteFile(cachemd5, []byte(d.MD5), FilePerm)
		if err != nil {
			log.Errorf("save into cache failed, error %+v", err)
		}
		err = nil
	}
	return
}

func lookupcache(name string, md5sum string) (path string, err error) {
	log.Debugf("lookupcache(name = %s, md5sum = %s)", name, md5sum)
	look := filepath.Join(config.GlobalConfig.CacheRoot, name)
	info, er := os.Stat(look)
	if er != nil {
		err = errors.Wrap(er, "error lookup cache")
		return
	}

	// Cache Should be a directory
	if !info.IsDir() {
		err = errors.New("error lookup cache: path is not a dir")
	}
	look = filepath.Join(look, "content")
	file, er := os.Open(look)
	if er != nil {
		err = errors.Wrap(er, "error lookup cache")
		return
	}
	defer file.Close()

	oldfile, er := ioutil.ReadFile(look)

	if er != nil {
		err = errors.Wrap(er, "error lookup cache")
		return
	}
	oldmd5 := md5.Sum(oldfile)

	if fmt.Sprintf("%x", oldmd5) != md5sum {
		err = errors.New("error lookup cache, md5sum do not match")
		return
	}
	path = filepath.Join(config.GlobalConfig.CacheRoot, name, CacheContent)
	return
}
