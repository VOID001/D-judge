package request

import (
	"bytes"
	"context"
	"encoding/base64"
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
	log.Debugf("Do(%v %v %v %v %v %v)", ctx, method, URL, data, ctype, respdata)
	req := new(http.Request)
	URL = config.GlobalConfig.EndpointURL + URL
	log.Debugf("stared request method=%s URL=%s", method, URL)
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
			} else {
				err = errors.New(fmt.Sprintf("do request error: data invaid type %T", data))
				return
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

	if respdata != nil {
		err = dec.Decode(&respdata)

		if err == io.EOF {
			err = nil
			respdata = nil
			return
		}

		if err != nil {
			err = errors.Wrap(err, "json decode error")
			return
		}
		log.Debugf("Decoded data %+v", respdata)
	}

	log.Debugf("done request method=%s URL=%s", method, URL)
	return
}

func JudgeError(ctx context.Context, errMsg error, jid int64) {
	info := make(url.Values)

	// Encode error to base64 string

	// For backward(domjudge) compability, set judge error as compile error
	data := base64.StdEncoding.EncodeToString([]byte(errMsg.Error()))
	info["compile_success"] = []string{"0"}
	info["output_compile"] = []string{data}
	info["judgehost"] = []string{config.GlobalConfig.HostName}

	err := Do(ctx, http.MethodPut, fmt.Sprintf("/judgings/%d", jid), info, TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "put Judging Errors error")
		log.Error(err)
	}
	return
}

func CompileError(ctx context.Context, compileErr error, jid int64) (err error) {
	info := make(url.Values)

	// Encode error to base64 string

	// For backward(domjudge) compability, set judge error as compile error
	data := base64.StdEncoding.EncodeToString([]byte(compileErr.Error()))
	info["compile_success"] = []string{"0"}
	info["output_compile"] = []string{data}
	info["judgehost"] = []string{config.GlobalConfig.HostName}

	err = Do(ctx, http.MethodPut, fmt.Sprintf("/judgings/%d", jid), info, TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "put Compile Errors error")
		return
	}
	return

}

func CompileOK(ctx context.Context, jid int64) (err error) {
	info := make(url.Values)

	info["compile_success"] = []string{"1"}
	info["output_compile"] = []string{""}
	info["judgehost"] = []string{config.GlobalConfig.HostName}

	err = Do(ctx, http.MethodPut, fmt.Sprintf("/judgings/%d", jid), info, TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "put Compile OK error")
		return
	}
	return

}

func PostResult(ctx context.Context, result config.RunResult) (err error) {
	info := make(url.Values)

	info["judgingid"] = []string{fmt.Sprintf("%d", result.JudgingID)}
	info["testcaseid"] = []string{fmt.Sprintf("%d", result.TestcaseID)}
	info["runresult"] = []string{result.RunResult}
	info["runtime"] = []string{fmt.Sprintf("%lf", result.RunTime)}
	info["judgehost"] = []string{config.GlobalConfig.HostName}
	info["output_run"] = []string{base64.StdEncoding.EncodeToString([]byte(result.OutputRun))}
	info["output_error"] = []string{base64.StdEncoding.EncodeToString([]byte(result.OutputError))}
	info["output_system"] = []string{base64.StdEncoding.EncodeToString([]byte(result.OutputSystem))}
	info["output_diff"] = []string{base64.StdEncoding.EncodeToString([]byte(result.OutputDiff))}

	err = Do(ctx, http.MethodPost, "/judging_runs", info, TypeForm, nil)
	if err != nil {
		err = errors.Wrap(err, "Post result error")
		return
	}
	return
}
