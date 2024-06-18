package main

import "strings"

type domainFlag []string

func (i domainFlag) String() string {

	return strings.Join(i[:], ",")
}

func (i *domainFlag) Set(f string) error {

	*i = append(*i, f)
	return nil
}
