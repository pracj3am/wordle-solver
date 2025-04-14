package dict

import "strings"

var (
	Conv = map[rune]rune{
		'a': 'a',
		'b': 'b',
		'c': 'c',
		'd': 'd',
		'e': 'e',
		'f': 'f',
		'g': 'g',
		'h': 'h',
		'i': 'i',
		'j': 'j',
		'k': 'k',
		'l': 'l',
		'm': 'm',
		'n': 'n',
		'o': 'o',
		'p': 'p',
		'q': 'q',
		'r': 'r',
		's': 's',
		't': 't',
		'u': 'u',
		'v': 'v',
		'w': 'w',
		'x': 'x',
		'y': 'y',
		'z': 'z',
		'á': 'a',
		'č': 'c',
		'ď': 'd',
		'é': 'e',
		'ě': 'e',
		'í': 'i',
		'ň': 'n',
		'ó': 'o',
		'ř': 'r',
		'š': 's',
		'ť': 't',
		'ú': 'u',
		'ů': 'u',
		'ý': 'y',
		'ž': 'z',
	}
)

func StripDiacritic(w string) string {
	var b strings.Builder
	for _, r := range w {
		b.WriteRune(Conv[r])
	}
	return b.String()
}
