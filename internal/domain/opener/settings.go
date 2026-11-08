package opener

import (
	"fmt"

	"github.com/heyuuu/x/store"

	"cube/pkg/xstore"
)

// settingsSection settings.json 中 opener 域的节名。
const settingsSection = "openers"

// intentsSection settings.json 中打开意图域的节名（1038）：键 = intent，
// 值 = {defaultOpener?, openers?}。openers 节保持纯清单，意图与默认独立成节。
const intentsSection = "openerIntents"

type SpecsStore = *xstore.KeyedSliceStore[Spec]

func newSpecsStore(settingsFile string) SpecsStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Spec]{
		Name:  "opener",
		Store: store.SectionStore[[]Spec](settingsFile, settingsSection),
		KeyOf: specName,
		Normalize: func(spec Spec) (Spec, error) {
			if spec.Name == "" {
				return spec, fmt.Errorf("opener name 不得为空")
			}

			stubBaseURL := "https://example.com" // 占位 url，用于校验占位
			_, err := InitActionOpener(spec, nil, stubBaseURL)
			return spec, err
		},
	})
}
