package project

import (
	"fmt"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/console"
	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/util/pathkit"
)

const checkItemCloneRules = "clone-rules"

var allCheckItems = []string{checkItemCloneRules}

// cmd `project list`
var projectCheckCmd = &easycobra.Command{
	Use:   "check <options>...",
	Short: "检查项目(目前 options 有: clone-rules)，不传会检查所有项目",
	Run: func(options []string) error {
		if len(options) == 0 {
			options = allCheckItems
		}

		for _, option := range options {
			switch option {
			case checkItemCloneRules:
				checkCloneRules()
				break
			default:
				return fmt.Errorf("未支持的 option: %s", option)
			}
		}
		return nil
	},
}

// 过滤出所有不符合 cloneRules 的项目
func checkCloneRules() {
	service := app.Default().ProjectService()
	projects := service.Projects()

	var headers = []string{"Name", "Path", "预期 Path", "RepoUrl"}
	var rows [][]string
	for _, p := range projects {
		repoUrl := p.RepoUrl()
		if repoUrl == "" {
			continue
		}

		_, localPath, ok := service.MatchCloneRule(repoUrl)

		// 有预期本地路径，但和目前实际路径不符合的情况下
		if ok && p.Path() != localPath {
			rows = append(rows, []string{
				p.Name(),
				pathkit.PrettyPath(p.Path()),
				pathkit.PrettyPath(localPath),
				repoUrl,
			})
		}
	}

	if len(rows) == 0 {
		fmt.Printf("> 没有不符合 clone rules 的项目\n")
		return
	}

	fmt.Printf("> 不符合 clone rules 的项目 %d 个:\n", len(rows))

	console.PrintTable(headers, rows)
}
