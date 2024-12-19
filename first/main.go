package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"unicode/utf8"

	"../dict"
	"../progress"
)

const (
	size = 5
)

type WeightedWord struct {
	Word   string
	Weight float64
}

type ByWeight []WeightedWord

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].Weight < a[j].Weight }

type LuckStat struct {
	Histogram map[int]int
	Sum       float64
}

type Tip struct {
	Word        string
	Left        int
	LeftNotUsed int
	Luck        *float64
	Skill       *int
}

var letters = []rune{
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
}

func LoadLuck(filePath string) (map[string]*LuckStat, error) {
	luck := make(map[string]*LuckStat)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	err = dec.Decode(&luck)
	if err != nil {
		return nil, err
	}

	return luck, nil
}

func CalculateOdds(
	word string,
	all []string,
	history map[string]bool,
	words dict.Dictionary,
	ppr *progress.Progress,
) (
	float64, *LuckStat,
) {
	var sum float64
	var luck LuckStat
	var count int

	luck.Histogram = make(map[int]int)

	for _, w := range all {
		if history[w] {
			continue
		}

		var counter int

		if w != word {
			pr := ppr.Clone()
			pr.ResetRound()

			pr.Guess(word, w)
			/*
				if word == "hubka" {
					var list []string
					_, counter, list = pr.WordsLeft(words, true)
					fmt.Printf("%s + %s: ", word, w)
					fmt.Println(list)

				} else {
			*/
			_, counter, _ = pr.WordsLeft(words, false)
		}

		luck.Histogram[counter]++
		luck.Sum++
		sum += float64(counter)
		count++
	}

	return sum / float64(count), &luck
}

func makeString(word []rune) string {
	var buf []byte
	for _, l := range word {
		buf = utf8.AppendRune(buf, l)
	}
	return string(buf)
}

func AppendTip(tips []Tip, word string, counter, counterNotUsed int, luck map[string]*LuckStat, skill map[string]*int) []Tip {
	tip := Tip{
		Word:        word,
		Left:        counter,
		LeftNotUsed: counterNotUsed,
	}

	if wordLuck, found := luck[word]; found {
		var sumBetter int
		for histLeft, histCount := range wordLuck.Histogram {
			if histLeft <= counterNotUsed {
				sumBetter += histCount
			}
		}
		luck := 100 - 100*float64(sumBetter)/(wordLuck.Sum)
		tip.Luck = &luck
		tip.Skill = skill[word]
	}

	return append(tips, tip)
}

func LoadDictionary(filePath string) ([]string, error) {
	words := make([]string, 2000)

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
	history, err := dict.LoadHistory("used.txt")
	if err != nil {
		fmt.Println("loading history failed", err)
		os.Exit(1)
	}

	words, err := dict.LoadDictionary("db.txt", history)
	if err != nil {
		fmt.Println("loading words failed", err)
		os.Exit(1)
	}

	all, err := LoadDictionary("db.txt")
	if err != nil {
		fmt.Println("loading all words failed", err)
		os.Exit(1)
	}

	luck := make(map[string]*LuckStat)
	skill := make(map[string]*int)
	progress := progress.NewProgress(size, letters)

	for i := 1; i <= 1; i++ {
		wordsLeftWeighted := make([]WeightedWord, len(all))
		for j, w := range all {
			odds, wordLuck := CalculateOdds(w, all, history, words, progress)
			wordsLeftWeighted[j] = WeightedWord{w, odds}
			luck[w] = wordLuck

		}
		sort.Sort(ByWeight(wordsLeftWeighted))

		for _, w := range wordsLeftWeighted {
			if history[w.Word] {
				w.Word += " *** "
			}
			fmt.Printf("%s %f\n", w.Word, w.Weight)
		}
		fmt.Println("")

		tmpSkill := make([]struct {
			w  string
			sk int
		}, len(wordsLeftWeighted))

		sk := 0
		wg := wordsLeftWeighted[0].Weight

		for i, w := range wordsLeftWeighted {
			if w.Weight > wg {
				sk++
				wg = w.Weight
			}
			tmpSkill[i].w = w.Word
			tmpSkill[i].sk = sk
		}

		maxSk := sk
		skill = make(map[string]*int)
		for _, s := range tmpSkill {
			sk := 0
			if maxSk > 0 {
				sk = (maxSk - s.sk) * 100 / maxSk
			}
			skill[s.w] = &sk
		}
	}

	f, err := os.OpenFile("luck.gob", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(luck)

	if err != nil {
		log.Fatal(err)
	}
	err = enc.Encode(skill)

	if err != nil {
		log.Fatal(err)
	}
}
