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
type AccountStore = *xstore.KeyedSliceStore[Account]

func newForgesStore(settingsFile string) ForgesStore {
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
	})
}

func accountKey(forgeHost, username string) string {
	return NormalizeUsername(username) + "@" + NormalizeHost(forgeHost)
}

func newAccountsStore(settingsFile string, forgesStore ForgesStore) AccountStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[Account]{
		Name:  "forge account",
		Store: store.SectionStore[[]Account](settingsFile, accountsSection),
		KeyOf: func(a Account) string { return accountKey(a.ForgeHost, a.Username) },
		NormalizeWithExists: func(a Account, exist *Account) (Account, error) {
			a.ForgeHost = NormalizeHost(a.ForgeHost)
			a.Username = NormalizeUsername(a.Username)
			if a.ForgeHost == "" {
				return a, fmt.Errorf("account 所属 forge host 不得为空")
			}
			if a.Username == "" {
				return a, fmt.Errorf("account username 不得为空")
			}

			if a.Token == TokenMasked {
				if exist == nil {
					return a, fmt.Errorf("account %s@%s 不存在，token 不能提交掩码占位值", a.Username, a.ForgeHost)
				}
				a.Token = exist.Token
			}

			f, ok := forgesStore.Find(a.ForgeHost)
			if !ok {
				return a, fmt.Errorf("account 所属 forge 未配置: %s", a.ForgeHost)
			}
			if f.Kind == KindGeneric {
				return a, fmt.Errorf("forge %s 是 generic 类型（无 API），不支持配置 account", a.ForgeHost)
			}
			return a, nil
		},
	})
}
