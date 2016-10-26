package config

type SystemConfig struct {
	HostName         string `toml:"host_name"`
	EndpointName     string `toml:"endpoint_name"`
	EndpointURL      string `toml:"endpoint_url"`
	MaxCacheSize     int    `toml:"max_cache_size"`
	EndpointPassword string `toml:"endpoint_password"`
	JudgeRoot        string `toml:"judge_root"`
	DockerImage      string `toml:"docker_image"`
	CacheRoot        string `toml:"cache_root"`
}
