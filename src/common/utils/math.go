package utils

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
