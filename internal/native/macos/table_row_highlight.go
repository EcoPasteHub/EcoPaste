//go:build darwin

package macos

import (
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/ebitengine/purego/objc"
)

// highlightTokens splits a search query into the terms to highlight. The
// backend matches every whitespace token independently, so the visible text
// is highlighted the same way.
func highlightTokens(query string) []string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(fields))
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		key := strings.ToLower(field)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tokens = append(tokens, field)
	}
	return tokens
}

// highlightRanges returns the UTF-16 ranges of text that match any token,
// merging overlapping matches so adjacent highlights read as one run.
// AppKit measures attribute ranges in UTF-16 code units, so the rune
// positions are converted before they reach an attributed string.
func highlightRanges(text string, tokens []string) []NSRange {
	if text == "" || len(tokens) == 0 {
		return nil
	}
	runes := []rune(text)
	lower := make([]rune, len(runes))
	for i, r := range runes {
		lower[i] = unicode.ToLower(r)
	}
	matched := make([]bool, len(runes))
	for _, token := range tokens {
		needle := []rune(token)
		if len(needle) == 0 || len(needle) > len(lower) {
			continue
		}
		for i := range needle {
			needle[i] = unicode.ToLower(needle[i])
		}
		for start := 0; start+len(needle) <= len(lower); start++ {
			ok := true
			for k := range needle {
				if lower[start+k] != needle[k] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			for k := range needle {
				matched[start+k] = true
			}
		}
	}

	offsets := make([]int, len(runes)+1)
	for i, r := range runes {
		offsets[i+1] = offsets[i] + utf16.RuneLen(r)
	}
	var ranges []NSRange
	for i := 0; i < len(matched); {
		if !matched[i] {
			i++
			continue
		}
		j := i
		for j < len(matched) && matched[j] {
			j++
		}
		ranges = append(ranges, NSRange{
			Location: uint64(offsets[i]),
			Length:   uint64(offsets[j] - offsets[i]),
		})
		i = j
	}
	return ranges
}

// highlightedRowString builds an attributed string that paints the matched
// keyword ranges with a soft highlighter background. It returns 0 when there
// is nothing to highlight, so callers fall back to a plain string value and
// keep the label's own font and color handling. view drives the light/dark
// highlight palette.
func highlightedRowString(text string, tokens []string, font, color, view objc.ID) objc.ID {
	ranges := highlightRanges(text, tokens)
	if len(ranges) == 0 {
		return 0
	}
	attr := objc.ID(class("NSMutableAttributedString")).Send(sel("alloc")).
		Send(sel("initWithString:"), nsString(text))
	if attr == 0 {
		return 0
	}
	full := NSRange{Length: uint64(objc.Send[uint](attr, sel("length")))}
	if font != 0 {
		attr.Send(sel("addAttribute:value:range:"), nsString("NSFont"), font, full)
	}
	if color != 0 {
		attr.Send(sel("addAttribute:value:range:"), nsString("NSColor"), color, full)
	}
	background, foreground := historyRowHighlightColors(view)
	for _, r := range ranges {
		attr.Send(sel("addAttribute:value:range:"), nsString("NSBackgroundColor"), background, r)
		attr.Send(sel("addAttribute:value:range:"), nsString("NSColor"), foreground, r)
	}
	return attr.Send(sel("autorelease"))
}

// historyRowHighlightColors is a muted highlighter marker rather than the
// saturated system find color: an opaque cream in light mode and a warm gold
// in dark mode, both paired with near-black text. Being opaque keeps the
// marker readable on the accent-colored selected card instead of blending
// into it.
func historyRowHighlightColors(view objc.ID) (background, foreground objc.ID) {
	if appearanceIsDark(view) {
		return sRGBColor(0.80, 0.68, 0.28), sRGBColor(0.12, 0.10, 0.02)
	}
	return sRGBColor(1.0, 0.945, 0.66), sRGBColor(0.12, 0.10, 0.02)
}

func sRGBColor(red, green, blue float64) objc.ID {
	return objc.ID(classNSColor).Send(sel("colorWithSRGBRed:green:blue:alpha:"), red, green, blue, 1.0)
}

// setRowLabelText applies a label's text, color and keyword highlight.
func setRowLabelText(label objc.ID, text string, tokens []string, font, color objc.ID) {
	label.Send(sel("setTextColor:"), color)
	if attr := highlightedRowString(text, tokens, font, color, label); attr != 0 {
		label.Send(sel("setAttributedStringValue:"), attr)
		return
	}
	label.Send(sel("setStringValue:"), nsString(text))
}
