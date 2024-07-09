package main

import (
	"strconv"
)

func strToValue(v string) any {
	switch v {
	case "null":
		return nil
	case "true":
		return true
	case "false":
		return false
	}

	vf, err := strconv.ParseFloat(v, 64)
	if err == nil {
		return vf
	}

	vi, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return vi
	}

	return v
}
