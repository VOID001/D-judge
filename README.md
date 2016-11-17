Docker Judge
====

Online Judgehost powered by Docker for NEUOJ

#### Installation

* You need to install go compiler first https://golang.org/ please use go1.7
* You need to have docker installed
* Run `go get -u -v github.com/VOID001/D-judge`
* cd to `$GOPATH/src/github.com/VOID001/D-judge/`
* Run `go build` then
* Create config.toml, you can copy one from config.toml.example


#### Run

* Configure the Judgehost specified configuration, more info can found in config.toml.example
* Run NEUOJ Server and start docker service
* Run `sudo ./D-judge` to start the judgehost

#### Contribution

* Please use pull request and github issue to contribute :)


#### FAQ

* Q: I got `json decode error: json: cannot unmarshal string into Go value of type int64` when running D-judge
* A: Please run `patch -p1 < 0001-Fix-compabability-issue-with-NEUOJ-Product-version.patch` in D-judge source root

* Q: I got `worker error: downloading testcase error: error processing download: request error status code 500 data`
* A: Please make sure you upload testcase to Server(NEUOJ)
