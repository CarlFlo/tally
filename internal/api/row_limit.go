package api

import (
	"strconv"
)

func rowLimit(raw string) (int, error) {
	if raw == "" {
		return 20, nil
	}
	n, e := strconv.Atoi(raw)
	if e != nil || (n != 20 && n != 50 && n != 100) {
		return 0, bad("choose 20, 50, or 100 rows")
	}
	return n, nil
}
