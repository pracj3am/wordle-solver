package main

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"regexp"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	size = 5
)

func LoadHistory(filePath string) ([]string, error) {
	words := make([]string, 0)

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
		words = append(words, word)
	}

	return words, s.Err()
}

func LoadDictionary(filePath string) ([]string, error) {
	words := make([]string, 0)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		words = append(words, s.Text())
	}

	return words, s.Err()
}

func main() {
	history, err := LoadHistory("used.txt")
	if err != nil {
		fmt.Println("loading history failed", err)
		os.Exit(1)
	}

	words, err := LoadDictionary("db.txt")
	if err != nil {
		fmt.Println("loading words failed", err)
		os.Exit(1)
	}

	for i, w := range history {
		counts := make(map[byte]int)
		c0 := make(map[byte]int)
		letters := make(map[byte][]int)
		allowed := make(map[int]map[byte]bool)

		for i := 0; i < 5; i++ {
			counts[w[i]]++
			c0[w[i]] = 0
			letters[w[i]] = append(letters[w[i]], i)
			allowed[i] = make(map[byte]bool)
			for j := 0; j < 5; j++ {
				allowed[i][w[j]] = true
			}
		}

		for l, pos := range letters {
			for _, p := range pos {
				allowed[p][l] = false
			}
		}

		var reParts [5][]byte

		for i, all := range allowed {
			for l, ok := range all {
				if ok {
					reParts[i] = append(reParts[i], l)
				}
			}
		}

		reStr := fmt.Sprintf("^[%s][%s][%s][%s][%s]$",
			reParts[0],
			reParts[1],
			reParts[2],
			reParts[3],
			reParts[4])

		re := regexp.MustCompile(reStr)

		for _, word := range words {
			if re.Match([]byte(word)) {
				c1 := maps.Clone(c0)
				for i := 0; i < 5; i++ {
					c1[word[i]]++
				}
				ok := true
				for l, c := range c1 {
					if c != counts[l] {
						ok = false
						break
					}
				}

				if ok {
					fmt.Println(i, w, word)
				}
			}
		}
	}
}
