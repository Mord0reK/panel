package integrations

import "sort"

var registered = []Service{
	adGuardHomeService{},
	jellyfinService{},
	tailscaleService{},
	cloudflareService{},
}

func ListDefinitions() []Definition {
	defs := make([]Definition, 0, len(registered))
	for _, svc := range registered {
		defs = append(defs, svc.Definition())
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Key < defs[j].Key
	})

	return defs
}

func GetDefinition(serviceKey string) (Definition, bool) {
	for _, svc := range registered {
		def := svc.Definition()
		if def.Key == serviceKey {
			return def, true
		}
	}

	return Definition{}, false
}

func GetService(serviceKey string) (Service, bool) {
	for _, svc := range registered {
		if svc.Definition().Key == serviceKey {
			return svc, true
		}
	}

	return nil, false
}
