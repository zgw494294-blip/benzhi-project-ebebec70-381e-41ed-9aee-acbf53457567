package httpapi

import "strings"

func pathParts(path string) []string { return strings.Split(strings.Trim(path, "/"), "/") }
