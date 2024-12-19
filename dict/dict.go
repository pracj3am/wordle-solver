package dict

import (
	"bufio"
	"os"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type DictionaryWord struct {
	Word string
	Used bool
}
type Dictionary map[rune]map[rune]map[rune]map[rune]map[rune]*DictionaryWord

func LoadDictionary(filePath string, history map[string]bool) (Dictionary, error) {
	words := make(Dictionary)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Bytes()

		i := 0
		l1, s := utf8.DecodeRune(word[i:])
		i += s
		l2, s := utf8.DecodeRune(word[i:])
		i += s
		l3, s := utf8.DecodeRune(word[i:])
		i += s
		l4, s := utf8.DecodeRune(word[i:])
		i += s
		l5, s := utf8.DecodeRune(word[i:])

		if words[l1] == nil {
			words[l1] = make(map[rune]map[rune]map[rune]map[rune]*DictionaryWord)
		}
		if words[l1][l2] == nil {
			words[l1][l2] = make(map[rune]map[rune]map[rune]*DictionaryWord)
		}
		if words[l1][l2][l3] == nil {
			words[l1][l2][l3] = make(map[rune]map[rune]*DictionaryWord)
		}
		if words[l1][l2][l3][l4] == nil {
			words[l1][l2][l3][l4] = make(map[rune]*DictionaryWord)
		}
		if words[l1][l2][l3][l4][l5] == nil {
			w := string(word)
			words[l1][l2][l3][l4][l5] = &DictionaryWord{w, history[w]}
		}
	}

	return words, s.Err()
}

func LoadHistory(filePath string) (map[string]bool, error) {
	words := make(map[string]bool)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Text()
		word, _, _ = transform.String(t, word)
		words[word] = true
	}

	return words, s.Err()
}
