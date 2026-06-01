package shared

func NormalizeStyleScope(scope StyleScope) StyleScope {
	if scope == StyleScopeApp {
		return StyleScopeApp
	}
	return StyleScopeGlobal
}
