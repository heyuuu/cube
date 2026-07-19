package opener

import (
	"github.com/heyuuu/cube/config"
)

type Opener struct {
	name string // 应用名, 唯一标识符
	bin  string // 应用路径
}

func NewOpener(conf config.OpenerConfig) *Opener {
	return &Opener{
		name: conf.Name,
		bin:  conf.Bin,
	}
}

func (app *Opener) Name() string { return app.name }
func (app *Opener) Bin() string  { return app.bin }
