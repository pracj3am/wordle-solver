package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"

	"../dict"
	"../odds"
	pr "../progress"
)

type LuckStat struct {
	Histogram map[int]int
	Sum       float64
}

type Skills struct {
	Robot *odds.Skill
	Human *odds.Skill // human nezná použitý slova
}

func CalculateOdds(
	word string,
	all []dict.DictionaryWord,
	ppr *pr.Progress,
) (
	float64, float64, *LuckStat,
) {
	var sumNotUsed, sum float64
	var luck LuckStat
	var countNotUsed, count int

	luck.Histogram = make(map[int]int)

	for _, dw := range all {
		var counter, counterNotUsed int

		if dw.WithoutDiacritics != word {
			pr := ppr.Clone()
			pr.ResetRound()

			pr.Guess(word, dw.Word)
			counter, counterNotUsed, _ = pr.WordsLeft(false)

			if counterNotUsed == 0 && !dw.Used {
				panic(fmt.Sprintf("%s + %s: counter 0", word, dw.Word))
			}
		}

		sum += float64(counter)
		count++

		if !dw.Used {
			luck.Histogram[counterNotUsed]++
			luck.Sum++
			sumNotUsed += float64(counterNotUsed)
			countNotUsed++
		}
	}

	return sum / float64(count), sumNotUsed / float64(countNotUsed), &luck
}

func LoadSolutions(filePath string, history map[string]bool) ([]dict.DictionaryWord, error) {
	words := make([]dict.DictionaryWord, 0, 2000)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	for s.Scan() {
		words = append(words, dict.DictionaryWord{
			Word:              s.Text(),
			WithoutDiacritics: dict.StripDiacritic(s.Text()),
			Used:              history[s.Text()],
		})
	}
	return words, s.Err()
}

func LoadDictionary(filePath string) ([]string, error) {
	words := make([]string, 0, 2000)
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
	history, err := dict.LoadHistory("../used.txt")
	if err != nil {
		fmt.Println("loading history failed", err)
		os.Exit(1)
	}

	words, err := dict.LoadDictionary("../db-hacky.txt", history)
	if err != nil {
		fmt.Println("loading words failed", err)
		os.Exit(1)
	}

	solutions, err := LoadSolutions("../db-hacky.txt", history)
	if err != nil {
		fmt.Println("loading all words failed", err)
		os.Exit(1)
	}

	all, err := LoadDictionary("../db.txt")
	if err != nil {
		fmt.Println("loading all words failed", err)
		os.Exit(1)
	}

	luck := make(map[string]*LuckStat)
	skillHuman := make(map[string]*odds.Skill)
	skillRobot := make(map[string]*odds.Skill)
	progress := pr.NewProgress(5, words)

	for i := 1; i <= 1; i++ {
		wordsLeftRobotWeighted := make([]odds.WeightedWord, len(all))
		wordsLeftHumanWeighted := make([]odds.WeightedWord, len(all))

		for j, w := range all {
			oddsHuman, oddsRobot, wordLuck := CalculateOdds(w, solutions, progress)
			wordsLeftRobotWeighted[j] = odds.WeightedWord{w, oddsRobot}
			wordsLeftHumanWeighted[j] = odds.WeightedWord{w, oddsHuman}
			luck[w] = wordLuck

			fmt.Printf("%d/%d\n", j, len(all))
		}

		sort.Sort(odds.ByWeight(wordsLeftHumanWeighted))
		skillHuman = odds.CalculateSkill(wordsLeftHumanWeighted)

		sort.Sort(odds.ByWeight(wordsLeftRobotWeighted))
		skillRobot = odds.CalculateSkill(wordsLeftRobotWeighted)

		for _, w := range wordsLeftRobotWeighted {
			if history[w.Word] {
				w.Word += " *** "
			}
			fmt.Printf("%s %f\n", w.Word, w.Weight)
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

	err = enc.Encode(skillRobot)
	if err != nil {
		log.Fatal(err)
	}

	err = enc.Encode(skillHuman)
	if err != nil {
		log.Fatal(err)
	}
}
