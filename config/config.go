package config

type Config struct {
	Log     LogConfig      `json:"log"`
	Project ProjectConfig  `json:"project"`
	Openers []OpenerConfig `json:"openers"`
}

type LogConfig struct {
	Path   string `json:"path"`
	Level  string `json:"level"`
	Format string `json:"format"`
}

type ProjectConfig struct {
	Scan  []ScanRuleConfig  `json:"scan"`
	Clone []CloneRuleConfig `json:"clone"`
}

type ScanRuleConfig struct {
	Group    string `json:"group"`
	Path     string `json:"path"`
	MaxDepth int    `json:"maxDepth"`
}

type CloneRuleConfig struct {
	RepoHost   string `json:"repoHost"`
	RepoPrefix string `json:"repoPrefix"`
	LocalPath  string `json:"localPath"`
}

type OpenerConfig struct {
	Name string `json:"name"`
	Bin  string `json:"bin"`
}
