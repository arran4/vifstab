package rules

import "sync"

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Rule)
)

func Register(r Rule) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[r.Name()] = r
}

func GetAllRules() []Rule {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var rules []Rule
	for _, r := range registry {
		rules = append(rules, r)
	}
	return rules
}
