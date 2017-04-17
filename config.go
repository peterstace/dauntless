package main

import "regexp"

type Config struct {
	WrapPrefix string
	BisectMask *regexp.Regexp
}
