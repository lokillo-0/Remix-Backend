package utils

func StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	categoryMap := make(map[string]bool)
	for _, v := range a {
		categoryMap[v] = true
	}
	for _, v := range b {
		if !categoryMap[v] {
			return false
		}
	}
	return true
}
