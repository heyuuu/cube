package template

import (
	"strings"

	"github.com/heyuuu/x/store"

	"cube/pkg/xstore"
)

// settings.json 中的模板源节名。
const tplSourcesSection = "tplSources"

type TplSourcesStore = *xstore.KeyedSliceStore[TplSource]

func newTplSourcesStore(settingsFile string) TplSourcesStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[TplSource]{
		Name:  "模板源",
		Store: store.SectionStore[[]TplSource](settingsFile, tplSourcesSection),
		KeyOf: func(src TplSource) string { return strings.TrimSpace(src.Name) },
		Normalize: func(src TplSource) (TplSource, error) {
			src.Name = strings.TrimSpace(src.Name)
			src.RepoURL = strings.TrimSpace(src.RepoURL)
			return src, ValidateTplSource(src)
		},
	})
}
