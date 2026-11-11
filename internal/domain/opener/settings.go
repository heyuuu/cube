package opener

import (
	"github.com/heyuuu/x/store"

	"cube/pkg/xstore"
)

// openersSection settings.json 中 opener 域的节名。
const openersSection = "openers"

// intentsSection settings.json 中打开意图域的节名（1038）：键 = intent，
// 值 = {defaultOpener?, openers?}。openers 节保持纯清单，意图与默认独立成节。
const intentsSection = "openerIntents"

type OpenersStore = *xstore.KeyedSliceStore[Opener]

func newOpenersStore(settingsFile string) OpenersStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Opener]{
		Name:      "opener",
		Store:     store.SectionStore[[]Opener](settingsFile, openersSection),
		KeyOf:     func(o Opener) string { return o.Name },
		Normalize: validateOpener,
	})
}
