package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	"./dict"
	pr "./progress"
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

type Skill struct {
	Relative   int
	Difficulty int
}

type Skills struct {
	Robot *Skill
	Human *Skill // human nezná použitý slova
}

type Tip struct {
	Word        string
	Left        int
	LeftNotUsed int
	Luck        *float64
	Skill       Skills
}

func LoadLuck(filePath string) (map[string]*LuckStat, map[string]*Skill, error) {
	luck := make(map[string]*LuckStat)
	oldSkill := make(map[string]*int)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	err = dec.Decode(&luck)
	if err != nil {
		return nil, nil, err
	}
	err = dec.Decode(&oldSkill)
	if err != nil {
		return nil, nil, err
	}

	skill := make(map[string]*Skill)
	for w, sk := range oldSkill {
		skill[w] = &Skill{Relative: *sk, Difficulty: -1}
	}

	return luck, skill, nil
}

func CalculateOdds(
	word string,
	all []string,
	history map[string]bool,
	words dict.Dictionary,
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

			/*
				if word == "jehne" {
					var list []string
					counter, counterNotUsed, list = pr.WordsLeft(words, true)
					fmt.Printf("%s + %s: ", word, w)
					fmt.Println(list)

				} else {
			*/

			counter, counterNotUsed, _ = pr.WordsLeft(words, false)
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

func makeString(word []rune) string {
	var buf []byte
	for _, l := range word {
		buf = utf8.AppendRune(buf, l)
	}
	return string(buf)
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

func AppendTip(
	tips []Tip,
	word string,
	counter, counterNotUsed int,
	luck map[string]*LuckStat,
	skillRobot map[string]*Skill,
	skillHuman map[string]*Skill,
) []Tip {
	tip := Tip{
		Word:        word,
		Left:        counter,
		LeftNotUsed: counterNotUsed,
	}

	if wordLuck, found := luck[word]; found {
		var sumBetter int
		var sumWorse int
		for histLeft, histCount := range wordLuck.Histogram {
			if histLeft <= counterNotUsed {
				sumBetter += histCount
			} else {
				sumWorse += histCount
			}
		}

		luck := -1.0
		if sumWorse > 0 {
			luck = 100 - 100*float64(sumBetter)/(wordLuck.Sum)
		}
		tip.Luck = &luck
	}
	tip.Skill = Skills{Human: skillHuman[word], Robot: skillRobot[word]}

	return append(tips, tip)
}

func PrintResuts(tips []Tip) {
	fmt.Println("Konec hry")
	for i := range tips {
		luck := "???%"
		if tips[i].Luck != nil {
			tLuck := *tips[i].Luck
			if tLuck < 0 {
				luck = " – "
			} else if tLuck < 10.0 {
				luck = fmt.Sprintf("%3.1f", tLuck)
			} else {
				luck = fmt.Sprintf("%3.0f", tLuck)
			}
		}
		skillRobot := "???"
		sk := tips[i].Skill.Robot
		if sk != nil {
			if sk.Difficulty == 0 {
				skillRobot = " – "
			} else {
				skillRobot = fmt.Sprintf("%3d", sk.Relative)
			}
		}

		skillHuman := "???"
		difficulty := "???"
		sk = tips[i].Skill.Human
		if tips[i].Skill.Human != nil {
			if sk.Difficulty == 0 {
				skillHuman = " – "
			} else {
				skillHuman = fmt.Sprintf("%3d", sk.Relative)
			}
			if sk.Difficulty >= 0 {
				difficulty = fmt.Sprintf("%3d", sk.Difficulty)
			}
		}
		fmt.Printf("%s 📶%s 🤖%s 🧠%s 🎲%s → %d/%d left\n",
			strings.ToUpper(tips[i].Word),
			difficulty, skillRobot, skillHuman, luck,
			tips[i].Left, tips[i].LeftNotUsed)
	}
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

	skillHuman := make(map[string]*Skill)
	luck, skillRobot, err := LoadLuck("luck.gob")
	if err != nil {
		fmt.Println("loading luck failed", err)
		os.Exit(1)
	}

	tips := make([]Tip, 0)

	interruptedCh := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGINT)
	signal.Notify(interruptedCh, syscall.SIGINT)

	go func() {
		<-interruptedCh
		PrintResuts(tips)
		os.Exit(0)
	}()

	progress := pr.NewProgress(size)

	stdIn := bufio.NewReader(os.Stdin)

	for i := 1; i <= 6; i++ {
		word := make([]rune, 5)
		progress.ResetRound()

		fmt.Printf("Tah č. %d\n", i)
		fmt.Printf("Slovo:\n")
		for j := 0; j < size; j++ {
			word[j], _, err = stdIn.ReadRune()
			if err != nil {
				fmt.Println("\n", err)
				os.Exit(2)
			}
		}
		stdIn.ReadLine() // read end of line

		fmt.Println("Označ zelenou(+), modrou(*), oranžovou(.) a šedou( ):")
		for j := 0; j < size; j++ {
			r, _, err := stdIn.ReadRune()
			if err != nil {
				fmt.Println("\n", err)
				os.Exit(2)
			}

			switch r {
			case ' ':
				progress.Grey(j, word[j])
			case '.':
				progress.Orange(j, word[j])
			case '*':
				progress.GreenOrange(j, word[j])
			case '+':
				progress.Green(j, word[j])
			default:
				fmt.Println("\nInvalid character")
				os.Exit(2)
			}
			word[j] = pr.Conv(word[j]) // odstranění diakritiky
		}
		stdIn.ReadLine() // read end of line

		guessedWord := makeString(word)

		counter, counterNotUsed, wordsLeft := progress.WordsLeft(words, true)

		if counter == 1 && pr.StripDiacritic(wordsLeft[0]) == guessedWord {
			counter = 0
			counterNotUsed = 0
		}

		fmt.Printf("\nZbývá %d slov\n\n", counter)

		tips = AppendTip(tips, guessedWord, counter, counterNotUsed, luck, skillRobot, skillHuman)

		if counter == 0 {
			break
		}

		if counter < 1000 {
			// jinak se to počítá moc dlouho
			wordsLeftRobotWeighted := make([]WeightedWord, len(wordsLeft))
			wordsLeftHumanWeighted := make([]WeightedWord, len(wordsLeft))

			for j, slovo := range wordsLeft {
				w := pr.StripDiacritic(slovo)
				oddsHuman, oddsRobot, wordLuck := CalculateOdds(w, wordsLeft, history, words, progress)
				wordsLeftRobotWeighted[j] = WeightedWord{slovo, oddsRobot}
				wordsLeftHumanWeighted[j] = WeightedWord{slovo, oddsHuman}
				luck[w] = wordLuck
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

		fmt.Println("")
	}

	PrintResuts(tips)
}
