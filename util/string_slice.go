package util

func SliceToMap(slice []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, v := range slice {
		m[v] = struct{}{}
	}

	return m
}
