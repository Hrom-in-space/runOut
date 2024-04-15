package utils

func InSlice[T comparable](s []T, val T) bool {
	for _, item := range s {
		if item == val {
			return true
		}
	}

	return false
}
