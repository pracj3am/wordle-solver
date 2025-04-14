package dict

import (
	"bufio"
	"os"
	"unicode/utf8"
)

var Letters = [41]rune{
	'a',
	'b',
	'c',
	'd',
	'e',
	'f',
	'g',
	'h',
	'i',
	'j',
	'k',
	'l',
	'm',
	'n',
	'o',
	'p',
	'q',
	'r',
	's',
	't',
	'u',
	'v',
	'w',
	'x',
	'y',
	'z',
	'a',
	'c',
	'd',
	'e',
	'e',
	'i',
	'n',
	'o',
	'r',
	's',
	't',
	'u',
	'u',
	'y',
	'z',
}
var Písmena = [41]rune{
	'a',
	'b',
	'c',
	'd',
	'e',
	'f',
	'g',
	'h',
	'i',
	'j',
	'k',
	'l',
	'm',
	'n',
	'o',
	'p',
	'q',
	'r',
	's',
	't',
	'u',
	'v',
	'w',
	'x',
	'y',
	'z',
	'á',
	'č',
	'ď',
	'é',
	'ě',
	'í',
	'ň',
	'ó',
	'ř',
	'š',
	'ť',
	'ú',
	'ů',
	'ý',
	'ž',
}
var Indexes = map[rune]int{
	'a': 0,
	'b': 1,
	'c': 2,
	'd': 3,
	'e': 4,
	'f': 5,
	'g': 6,
	'h': 7,
	'i': 8,
	'j': 9,
	'k': 10,
	'l': 11,
	'm': 12,
	'n': 13,
	'o': 14,
	'p': 15,
	'q': 16,
	'r': 17,
	's': 18,
	't': 19,
	'u': 20,
	'v': 21,
	'w': 22,
	'x': 23,
	'y': 24,
	'z': 25,
	'á': 26,
	'č': 27,
	'ď': 28,
	'é': 29,
	'ě': 30,
	'í': 31,
	'ň': 32,
	'ó': 33,
	'ř': 34,
	'š': 35,
	'ť': 36,
	'ú': 37,
	'ů': 38,
	'ý': 39,
	'ž': 40,
}

type DictionaryWord struct {
	Word string
	Used bool
}

type nextLetter struct {
	Next [41]*nextLetter
	Word *DictionaryWord
}
type Dictionary struct {
	First [41]*nextLetter
}

func LoadDictionary(filePath string, history map[string]bool) (*Dictionary, error) {
	var words Dictionary

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Bytes()

		i := 0
		l1, t := utf8.DecodeRune(word[i:])
		i += t
		l2, t := utf8.DecodeRune(word[i:])
		i += t
		l3, t := utf8.DecodeRune(word[i:])
		i += t
		l4, t := utf8.DecodeRune(word[i:])
		i += t
		l5, t := utf8.DecodeRune(word[i:])

		i1, i2, i3, i4, i5 := Indexes[l1], Indexes[l2], Indexes[l3], Indexes[l4], Indexes[l5]

		if words.First[i1] == nil {
			words.First[i1] = new(nextLetter)
		}
		if words.First[i1].Next[i2] == nil {
			words.First[i1].Next[i2] = new(nextLetter)
		}
		if words.First[i1].Next[i2].Next[i3] == nil {
			words.First[i1].Next[i2].Next[i3] = new(nextLetter)
		}
		if words.First[i1].Next[i2].Next[i3].Next[i4] == nil {
			words.First[i1].Next[i2].Next[i3].Next[i4] = new(nextLetter)
		}
		if words.First[i1].Next[i2].Next[i3].Next[i4].Next[i5] == nil {
			lastLetter := new(nextLetter)
			w := s.Text()
			lastLetter.Word = &DictionaryWord{w, history[w]}
			words.First[i1].Next[i2].Next[i3].Next[i4].Next[i5] = lastLetter
		}
	}

	return &words, s.Err()
}

func LoadHistory(filePath string) (map[string]bool, error) {
	words := make(map[string]bool)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Text()
		words[word] = true
	}

	return words, s.Err()
}
