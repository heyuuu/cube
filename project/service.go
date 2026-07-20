package project

import (
	"log"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/easycache"
	"github.com/heyuuu/cube/util/fuzzy"
	"github.com/heyuuu/cube/util/slicekit"
)

type Service struct {
	// scan
	scanRules []ScanRule                  // 项目扫描规则
	scanCache *easycache.Item[[]*Project] // 项目扫描的缓存
	// clone
	cloneRules []CloneRule // 项目 clone 规则
}

func NewService(conf config.ProjectConfig) *Service {
	scanRules := slicekit.Map(conf.Scan, func(r config.ScanRuleConfig) ScanRule {
		return ScanRule{
			Group:    r.Group,
			Path:     r.Path,
			MaxDepth: r.MaxDepth,
		}
	})
	cloneRules := slicekit.Map(conf.Clone, func(r config.CloneRuleConfig) CloneRule {
		return CloneRule{
			RepoHost:   r.RepoHost,
			RepoPrefix: r.RepoPrefix,
			LocalPath:  r.LocalPath,
		}
	})

	// 构建 service 实体
	s := &Service{
		scanRules:  scanRules,
		cloneRules: cloneRules,
	}
	s.scanCache = easycache.NewItem(s.loadProjects)
	return s
}

// -- getter --

func (s *Service) ScanRules() []ScanRule   { return s.scanRules }
func (s *Service) CloneRules() []CloneRule { return s.cloneRules }

// --- project 读操作 ---

func (s *Service) Projects() []*Project {
	return s.scanCache.Get()
}

func (s *Service) FindByPath(path string) *Project {
	for _, proj := range s.Projects() {
		if proj.Path() == path {
			return proj
		}
	}
	return nil
}

func (s *Service) FindByName(name string) *Project {
	for _, proj := range s.Projects() {
		if proj.Name() == name {
			return proj
		}
	}
	return nil
}

func (s *Service) Search(query string) []*Project {
	return fuzzy.MatchBy(query, s.Projects(), (*Project).Name, nil)
}

// --- scan 相关 ---

// 加载所有项目的实际逻辑
func (s *Service) loadProjects() []*Project {
	var result []*Project
	for _, rule := range s.scanRules {
		projects, err := ScanProjects(rule)
		if err != nil {
			log.Println(err)
		}
		result = append(result, projects...)
	}
	return result
}

// --- clone 相关 ---

func (s *Service) MatchCloneRule(repoUrl string) (rule CloneRule, localPath string, ok bool) {
	return MatchCloneRule(repoUrl, s.cloneRules)
}
