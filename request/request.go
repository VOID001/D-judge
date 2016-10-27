package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/VOID001/D-judge/config"
	"github.com/pkg/errors"
)

const (
	TypeForm = "application/x-www-form-urlencoded"
	TypeJSON = "application/json"
)

func Do(ctx context.Context, method string, URL string, data interface{}, ctype string, respdata interface{}) (err error) {
	var req *http.Request
	URL = config.GlobalConfig.EndpointURL + URL
	log.Infof("stared request method=%s URL=%s", method, URL)
	cli := &http.Client{}
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)

	// Get should not have body
	if method != http.MethodGet && data != nil {
		if ctype == TypeForm {
			if formdata, ok := data.(url.Values); ok {
				req, err = http.NewRequest(method, URL, bytes.NewBufferString(formdata.Encode()))
				if err != nil {
					err = errors.Wrap(err, "do request error")
					return
				}
			}
		} else if ctype == TypeJSON {
			enc.Encode(data)
			req, err = http.NewRequest(method, URL, &buf)
		} else {
			err = errors.New(fmt.Sprintf("do request error unsupported content-type %s", ctype))
			return err
		}
		req.Header.Add("Content-Type", ctype)
	} else {
		req, err = http.NewRequest(method, URL, nil)
	}

	req.Header.Add("X-Djudge-Hostname", config.GlobalConfig.HostName)
	req.SetBasicAuth(config.GlobalConfig.EndpointUser, config.GlobalConfig.EndpointPassword)

	resp, err := cli.Do(req)
	log.Debugf("request header is %+v", req.Header)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("request error method=%s URL=%s", method, URL))
		return
	}
	defer resp.Body.Close()

	tmpbuf := bytes.Buffer{}
	dec := json.NewDecoder(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		tmpbuf.ReadFrom(resp.Body)
		err = errors.New(fmt.Sprintf("request error status code %d data\n %s", resp.StatusCode, tmpbuf.String()))
		return
	}
	log.Debugf("Response Header %+v", resp.Header)
	log.Debugf("Response Header %s", tmpbuf.String())

	err = dec.Decode(&respdata)

	if err == io.EOF {
		err = nil
		return
	}
	if err != nil {
		err = errors.Wrap(err, "json decode error")
		return
	}
	log.Debugf("Decoded data %+v", respdata)

	log.Infof("done request method=%s URL=%s", method, URL)
	return
}
