package rules

import "sync"

var (
	registryMu  sync.RWMutex
	registry    = make(map[string]Rule)
	cachedRules []Rule
)

func Register(r Rule) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[r.Name()] = r
	cachedRules = nil
}

func GetAllRules() []Rule {
	registryMu.RLock()
	if cachedRules != nil {
		res := cachedRules
		registryMu.RUnlock()
		return res
	}
	registryMu.RUnlock()

	registryMu.Lock()
	defer registryMu.Unlock()
	if cachedRules != nil {
		return cachedRules
	}

	cachedRules = make([]Rule, 0, len(registry))
	for _, r := range registry {
		cachedRules = append(cachedRules, r)
	}

	return cachedRules
}
