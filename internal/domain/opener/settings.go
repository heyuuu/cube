package opener

import (
	"github.com/heyuuu/x/store"

	"cube/pkg/xstore"
)

// settingsSection settings.json 中 opener 域的节名。
const settingsSection = "openers"

// intentsSection settings.json 中打开意图域的节名（1038）：键 = intent，
// 值 = {defaultOpener?, openers?}。openers 节保持纯清单，意图与默认独立成节。
const intentsSection = "openerIntents"

type SpecsStore = *xstore.KeyedSliceStore[Opener]

func newSpecsStore(settingsFile string) SpecsStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Opener]{
		Name:      "opener",
		Store:     store.SectionStore[[]Opener](settingsFile, settingsSection),
		KeyOf:     func(o Opener) string { return o.Name },
		Normalize: validateOpener,
	})
}
