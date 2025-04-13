package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"

	"../dict"
	pr "../progress"
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

type Skill struct {
	Relative   int
	Difficulty int
}

type Skills struct {
	Robot *Skill
	Human *Skill // human nezná použitý slova
}

func CalculateOdds(
	word string,
	all []string,
	history map[string]bool,
	ppr *pr.Progress,
) (
	float64, float64, *LuckStat,
) {
	var sumNotUsed, sum float64
	var luck LuckStat
	var countNotUsed, count int

	luck.Histogram = make(map[int]int)

	for _, w := range all {
		var counter, counterNotUsed int

		if pr.StripDiacritic(w) != word {
			pr := ppr.Clone()
			pr.ResetRound()

			pr.Guess(word, w)
			counter, counterNotUsed, _ = pr.WordsLeft(false)
		}

		sum += float64(counter)
		count++

		if !history[w] {
			luck.Histogram[counterNotUsed]++
			luck.Sum++
			sumNotUsed += float64(counterNotUsed)
			countNotUsed++
		}
	}

	return sum / float64(count), sumNotUsed / float64(countNotUsed), &luck
}

func CalculateSkill(words []WeightedWord) map[string]*Skill {
	skill := make(map[string]*Skill)
	tmpSkill := make([]struct {
		w  string
		sk int
	}, len(words))

	sk := 0
	wg := words[0].Weight

	for i, w := range words {
		if w.Weight > wg {
			sk++
			wg = w.Weight
		}
		tmpSkill[i].w = w.Word
		tmpSkill[i].sk = sk
	}

	maxSk := sk
	for _, s := range tmpSkill {
		sk := 0
		if maxSk > 0 {
			sk = (maxSk - s.sk) * 100 / maxSk
		}
		w := pr.StripDiacritic(s.w)
		skill[w] = &Skill{Relative: sk, Difficulty: maxSk}
	}

	return skill
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
	history, err := dict.LoadHistory("used.txt")
	if err != nil {
		fmt.Println("loading history failed", err)
		os.Exit(1)
	}

	words, err := dict.LoadDictionary("db-hacky.txt", history)
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
	skillHuman := make(map[string]*Skill)
	skillRobot := make(map[string]*Skill)
	progress := pr.NewProgress(5, words)

	for i := 1; i <= 1; i++ {
		wordsLeftRobotWeighted := make([]WeightedWord, len(all))
		wordsLeftHumanWeighted := make([]WeightedWord, len(all))

		for j, slovo := range all {
			w := pr.StripDiacritic(slovo)
			oddsHuman, oddsRobot, wordLuck := CalculateOdds(w, all, history, progress)
			wordsLeftRobotWeighted[j] = WeightedWord{slovo, oddsRobot}
			wordsLeftHumanWeighted[j] = WeightedWord{slovo, oddsHuman}
			luck[w] = wordLuck

			fmt.Print(".")
		}

		sort.Sort(ByWeight(wordsLeftHumanWeighted))
		skillHuman = CalculateSkill(wordsLeftHumanWeighted)

		sort.Sort(ByWeight(wordsLeftRobotWeighted))
		skillRobot = CalculateSkill(wordsLeftRobotWeighted)

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
