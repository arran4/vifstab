package rules

import (
	"sort"
	"sync"
)

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
	sort.Slice(rules, func(i, j int) bool { return rules[i].Name() < rules[j].Name() })
	return rules
}
