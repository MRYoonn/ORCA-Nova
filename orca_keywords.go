package main

import "strings"

// ORCA 6 simple-input spelling differs from the literature/UI label.
func orcaFunctionalKeyword(name string) string {
 if strings.EqualFold(name,"M06-2X") { return "M062X" }
 return name
}
