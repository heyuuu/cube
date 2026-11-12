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

type IntentInfosStore = *xstore.KeyedSliceStore[IntentInfo]

func newOpenersStore(settingsFile string) OpenersStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Opener]{
		Name:      "opener",
		Store:     store.SectionStore[[]Opener](settingsFile, openersSection),
		KeyOf:     openerName,
		Normalize: validateOpener,
	})
}

func newIntentInfosStore(settingsFile string) IntentInfosStore {
	type Spec struct {
		DefaultOpener string   `json:"defaultOpener,omitempty"`
		Openers       []string `json:"openers,omitempty"`
	}

	// 内部存储，持久化存储为 map[Intent]Spec，内存中为 []IntentInfo
	// 此处仅数据结构转换，不在此校验
	s := store.WrapperStore[map[Intent]Spec, []IntentInfo](
		store.SectionStore[map[Intent]Spec](settingsFile, intentsSection),
		func(m map[Intent]Spec) ([]IntentInfo, error) {
			infos := make([]IntentInfo, 0, len(intentOrder))
			for _, intent := range intentOrder {
				spec := m[intent]
				infos = append(infos, IntentInfo{
					Intent:        intent,
					DefaultOpener: spec.DefaultOpener,
					Openers:       spec.Openers,
				})
			}
			return infos, nil
		},
		func(infos []IntentInfo) (map[Intent]Spec, error) {
			specs := make(map[Intent]Spec, len(infos))
			for _, info := range infos {
				if info.DefaultOpener == "" && len(info.Openers) == 0 {
					continue
				}
				specs[info.Intent] = Spec{
					DefaultOpener: info.DefaultOpener,
					Openers:       info.Openers,
				}
			}
			return specs, nil
		},
	)

	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[IntentInfo]{
		Name:  "openerIntents",
		Store: s,
		KeyOf: func(info IntentInfo) string { return string(info.Intent) },
	})
}
