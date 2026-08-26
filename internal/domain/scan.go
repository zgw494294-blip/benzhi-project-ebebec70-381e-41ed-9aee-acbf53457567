package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

type Pattern struct {
	Category string
	Re       *regexp.Regexp
}

var SensitivePatterns = []Pattern{
	{Category: "姓名", Re: regexp.MustCompile(`(?:姓名|联系人|我叫)[：:]?([\p{Han}]{2,4})`)},
	{Category: "联系方式", Re: regexp.MustCompile(`(?:1[3-9]\d{9}|\d{3,4}[- ]?\d{7,8}|[\w.+-]+@[\w.-]+\.[A-Za-z]{2,})`)},
	{Category: "精确地点", Re: regexp.MustCompile(`(?:北京市|上海市|广州市|深圳市)[\p{Han}]{1,12}(?:号|路|街|区)`)},
	{Category: "受限主题", Re: regexp.MustCompile(`(?:未成年人|医疗记录|政治避难|家庭暴力|性侵)`)},
}

func ScanText(segmentID, text string) []Finding {
	var out []Finding
	for _, p := range SensitivePatterns {
		for _, m := range p.Re.FindAllStringIndex(text, -1) {
			out = append(out, Finding{FindingID: DeterministicFindingID(segmentID, p.Category, m[0], m[1]), SegmentID: segmentID, Category: p.Category, StartOffset: m[0], EndOffset: m[1], Decision: "PENDING"})
		}
	}
	return out
}

func DeterministicFindingID(segmentID, category string, start, end int) string {
	sum := sha256.Sum256([]byte(segmentID + "|" + category + "|" + itoa(start) + "|" + itoa(end)))
	return "f-" + hex.EncodeToString(sum[:8])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func NewID(prefix string) string { return prefix + "-" + RandomToken() }
