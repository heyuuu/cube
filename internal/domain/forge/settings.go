package forge

import (
	"fmt"
	"strings"

	"github.com/heyuuu/x/store"

	"cube/internal/domain/icons"
	"cube/pkg/xstore"
)

// settings.json 中的 forge 域节名。
const forgesSection = "forges"

// settings.json 中的 account 域节名。
const accountsSection = "forgeAccounts"

type ForgesStore = *xstore.KeyedSliceStore[Forge]
type AccountsStore = *xstore.KeyedSliceStore[Account]

func newForgesStore(settingsFile string, onChange func([]Forge) error) ForgesStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Forge]{
		Name:  "forge 配置",
		Store: store.SectionStore[[]Forge](settingsFile, forgesSection),
		KeyOf: func(f Forge) string { return NormalizeHost(f.Host) },
		Normalize: func(f Forge) (Forge, error) {
			f.Host = NormalizeHost(f.Host)
			if err := ValidateHost(f.Host); err != nil {
				return f, err
			}
			if !ValidKind(f.Kind) {
				return f, fmt.Errorf("forge kind 未知: %q（合法值：%s）", f.Kind, strings.Join(Kinds(), " / "))
			}
			if err := icons.ValidateIcon(f.Icon); err != nil {
				return f, fmt.Errorf("forge icon 配置错误: %s", err)
			}
			return f, nil
		},
		OnChange: onChange,
	})
}

func accountKey(forgeHost, username string) string {
	return NormalizeUsername(username) + "@" + NormalizeHost(forgeHost)
}

func accountKeyOf(a Account) string {
	return accountKey(a.ForgeHost, a.Username)
}

func newAccountsStore(settingsFile string, onChange func([]Account) error) AccountsStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Account]{
		Name:      "forge account",
		Store:     store.SectionStore[[]Account](settingsFile, accountsSection),
		KeyOf:     accountKeyOf,
		Normalize: normalizeAccount,
		OnChange:  onChange,
	})
}
