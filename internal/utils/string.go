package utils

func SetupLookupMap(array []string) map[string]bool {
	m := make(map[string]bool, len(array))
	for _, v := range array {
		m[v] = true
	}
	return m
}
