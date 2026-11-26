package main

import (
	"regexp"
	"strings"
)

// Shared ORCA 6.0 and 6.1 spellings for the functionals exposed by the app.
// Keep the browser table in sync; ORCA 7 has not been qualified.
var orca6Functionals = map[string]string{
	"B3LYP": "B3LYP", "PBE0": "PBE0", "TPSSh": "TPSSh", "BP86": "BP86",
	"M06-2X": "M062X", "wB97X-D3": "wB97X-D3", "r2SCAN-3c": "r2SCAN-3c",
}

func orcaFunctionalKeyword(name string) string {
	for label, keyword := range orca6Functionals {
		if strings.EqualFold(name, label) || strings.EqualFold(name, keyword) {
			return keyword
		}
	}
	return name
}

var orcaSimpleInput = regexp.MustCompile(`(?m)^[\t ]*![^#\r\n]*`)
var orcaToken = regexp.MustCompile(`[^\s!]+`)

func normalizeORCAInput(input string) string {
	return orcaSimpleInput.ReplaceAllStringFunc(input, func(line string) string {
		return orcaToken.ReplaceAllStringFunc(line, func(token string) string {
			keyword := orcaFunctionalKeyword(token)
			if strings.EqualFold(keyword, token) {
				return token
			}
			return keyword
		})
	})
}

func normalizeORCARequest(req RunRequest) RunRequest {
	original := req.Files
	copied := false
	for i, f := range original {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".inp") {
			continue
		}
		if canonical := normalizeORCAInput(f.Content); canonical != f.Content {
			if !copied {
				copied = true
				req.Files = append([]JobFile(nil), original...)
			}
			req.Files[i].Content = canonical
		}
	}
	return req
}

func duplicateORCAFunctional(inp string) bool {
	count := 0
	for _, line := range orcaSimpleInput.FindAllString(inp, -1) {
		for _, token := range orcaToken.FindAllString(line, -1) {
			for _, keyword := range orca6Functionals {
				if strings.EqualFold(orcaFunctionalKeyword(token), keyword) {
					count++
				}
			}
		}
	}
	return count > 1
}
