package dict

import (
	"bufio"
	"os"
	"unicode/utf8"
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
		l1, t := utf8.DecodeRune(word[i:])
		i += t
		l2, t := utf8.DecodeRune(word[i:])
		i += t
		l3, t := utf8.DecodeRune(word[i:])
		i += t
		l4, t := utf8.DecodeRune(word[i:])
		i += t
		l5, t := utf8.DecodeRune(word[i:])

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
			w := s.Text()
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

	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Text()
		words[word] = true
	}

	return words, s.Err()
}
