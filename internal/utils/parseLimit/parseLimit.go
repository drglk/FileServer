package utils

import "strconv"

func ParseLimit(s string) int {
	limit, err := strconv.Atoi(s)
	if err != nil || limit <= 0 {
		return 0
	}

	return limit
}
