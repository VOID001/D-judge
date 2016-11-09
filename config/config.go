package config

var GlobalConfig SystemConfig

// Define Run results
const (
	ResTLE = "timelimit"
	ResWA  = "wrong-answer"
	ResAC  = "correct"
	ResCE  = "compiler-error"
	ResRE  = "run-error"
)

type SystemConfig struct {
	HostName         string `toml:"host_name"`
	EndpointUser     string `toml:"endpoint_user"`
	EndpointName     string `toml:"endpoint_name"`
	EndpointURL      string `toml:"endpoint_url"`
	MaxCacheSize     int    `toml:"max_cache_size"`
	EndpointPassword string `toml:"endpoint_password"`
	JudgeRoot        string `toml:"judge_root"`
	DockerImage      string `toml:"docker_image"`
	DockerServer     string `toml:"docker_server"`
	CacheRoot        string `toml:"cache_root"`
	RootMemory       int64  `toml:"root_mem"`
}

type JudgeInfo struct {
	SubmitID      int64  `json:"submitid,string"`
	ContestID     int64  `json:"cid"`
	TeamID        int64  `json:"teamid,string"`
	JudgingID     int64  `json:"judgingid,string"`
	ProblemID     int64  `json:"probid,string"`
	Language      string `json:"langid"`
	TimeLimit     int64  `json:"maxruntime,string"`
	MemLimit      int64  `json:"memlimit,string"`
	OutputLimit   int64  `json:"output_limit,string"`
	BuildZip      string `json:"compile_script"`
	BuildZipMD5   string `json:"compile_script_md5sum"`
	RunZip        string `json:"run"`
	RunZipMD5     string `json:"run_md5sum"`
	CompareZip    string `json:"compare"`
	CompareZipMD5 string `json:"compare_md5sum"`
	CompareArgs   string `json:"compare_args"`
}

type TestcaseInfo struct {
	TestcaseID   int64  `json:"testcaseid,string"`
	Rank         int64  `json:"rank"`
	ProblemID    int64  `json:"probid,string"`
	MD5SumInput  string `json:"md5sum_input"`
	MD5SumOutput string `json:"md5sum_output"`
}

type SubmissionInfo struct {
	info []SubmissionFileInfo `json:""`
}

type SubmissionFileInfo struct {
	FileName string `json:"filename"`
	Content  string `json:"contetn"`
}

type RunResult struct {
	JudgingID    int64
	TestcaseID   int64
	RunResult    string
	RunTime      float64
	OutputRun    string
	OutputError  string
	OutputSystem string
	OutputDiff   string
}
