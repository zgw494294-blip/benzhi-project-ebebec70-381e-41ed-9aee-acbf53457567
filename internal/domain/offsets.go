package domain

func ApplyReplacement(text string, start, end int, replacement string) (string, error) {
	if start < 0 || end < start || end > len(text) {
		return "", ErrInvalid
	}
	return text[:start] + replacement + text[end:], nil
}
func ApplyFindings(text string, fs []Finding) (string, []string, error) {
	out := text
	changes := []string{}
	for i := len(fs) - 1; i >= 0; i-- {
		f := fs[i]
		r := f.ReplacementText
		switch f.Decision {
		case "DELETE":
			r = ""
		case "GENERALIZE":
			r = "[已泛化]"
		case "KEEP":
			continue
		}
		var e error
		out, e = ApplyReplacement(out, f.StartOffset, f.EndOffset, r)
		if e != nil {
			return "", nil, e
		}
		changes = append(changes, f.FindingID)
	}
	return out, changes, nil
}
