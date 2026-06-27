// Package analyzer poskytuje analýzu odehrané hry (Wordle) jako knihovnu:
// pro každý tip spočítá zbývající slova, obtížnost, "IQ" a štěstí.
// Vše se počítá živě nad slovníkem (žádný předpočítaný luck.gob).
package analyzer

import (
	"sort"
	"strings"

	"github.com/pracj3am/wordle-solver/dict"
	"github.com/pracj3am/wordle-solver/odds"
	pr "github.com/pracj3am/wordle-solver/progress"
)

// oddsThreshold: nad tolik kandidátů se obtížnost/IQ nepočítá (kvadratický výpočet
// by byl moc pomalý) → 1. tah obvykle vyjde jako "–", stejně jako v referenci.
const oddsThreshold = 1000

// LuckStat = distribuce počtu zbylých možných odpovědí pro daný tip.
type LuckStat struct {
	Histogram map[int]int
	Sum       float64
}

// Row = výsledek analýzy jednoho tipu. -1 znamená "nedostupné" ("–").
type Row struct {
	Word        string  `json:"word"`
	Left        int     `json:"left"`        // všechna platná zbývající slova
	LeftAnswers int     `json:"leftAnswers"` // z toho možné odpovědi
	Difficulty  int     `json:"difficulty"`  // -1 = "–"
	IQ          int     `json:"iq"`          // 0..100, nebo -1
	Luck        float64 `json:"luck"`        // %, nebo -1
}

// Engine drží načtený slovník. Možné odpovědi (answers) mají Used=false,
// ostatní platná slova Used=true (nejsou možné odpovědi).
type Engine struct {
	dict *dict.Dictionary
}

func NewEngine(dictPath string, answers []string) (*Engine, error) {
	all, err := dict.LoadHistory(dictPath) // všechna slova ze souboru (s diakritikou)
	if err != nil {
		return nil, err
	}
	ans := make(map[string]bool, len(answers))
	for _, a := range answers {
		ans[dict.StripDiacritic(a)] = true
	}
	history := make(map[string]bool) // Used = NENÍ možná odpověď
	for w := range all {
		if !ans[dict.StripDiacritic(w)] {
			history[w] = true
		}
	}
	d, err := dict.LoadDictionary(dictPath, history)
	if err != nil {
		return nil, err
	}
	return &Engine{dict: d}, nil
}

// calcOdds = port reference CalculateOdds: simuluje tip "word" proti každé možné
// odpovědi z "all" a vrací průměrný počet zbylých slov (human=z odpovědí, robot=ze
// všech) a histogram štěstí.
func calcOdds(word *dict.DictionaryWord, all []*dict.DictionaryWord, base *pr.Progress) (human, robot float64, luck *LuckStat) {
	var sum, sumNotUsed float64
	var count, countNotUsed int
	luck = &LuckStat{Histogram: make(map[int]int)}
	for _, dw := range all {
		var counter, counterNotUsed int
		if dw.WithoutDiacritics != word.WithoutDiacritics {
			p := base.Clone()
			p.ResetRound()
			p.Guess(word.Word, dw.Word)
			counter, counterNotUsed, _ = p.WordsLeft(false)
		}
		sum += float64(counter)
		count++
		if !dw.Used { // dw je možná odpověď
			luck.Histogram[counterNotUsed]++
			luck.Sum++
			sumNotUsed += float64(counterNotUsed)
			countNotUsed++
		}
	}
	robot = sum / float64(count)
	if countNotUsed > 0 {
		human = sumNotUsed / float64(countNotUsed)
	}
	return
}

// luckPct = port reference: procento řešení, u nichž by tip dopadl hůř (vyšší = víc štěstí).
func luckPct(ls *LuckStat, counterNotUsed int) float64 {
	if ls == nil || ls.Sum == 0 {
		return -1
	}
	var sumBetter, sumWorse, countBetter int
	for histLeft, histCount := range ls.Histogram {
		if histLeft <= counterNotUsed {
			sumBetter += histCount
			if histLeft > 0 {
				countBetter++
			}
		} else {
			sumWorse += histCount
		}
	}
	if sumWorse > 0 || countBetter > 1 {
		return 100 - 100*float64(sumBetter)/ls.Sum
	}
	return -1
}

// Analyze: pro každý tip (základní písmena bez diakritiky) spočítá metriky.
// solution = denní slovo (může mít diakritiku); zpětnou vazbu odvodí progress.Guess.
// Věrně kopíruje smyčku referenčního CLI: metriky pro tip se počítají na konci
// předchozího kola nad tehdy zbývajícími slovy.
func (e *Engine) Analyze(guesses []string, solution string) []Row {
	progress := pr.NewProgress(5, e.dict)
	rows := make([]Row, 0, len(guesses))

	// luckMap/skillMap = metriky pro AKTUÁLNÍ tip (spočítané na konci minulého kola)
	var luckMap map[string]*LuckStat
	var skillMap map[string]*odds.Skill

	// 1. tip: štěstí nad plným fondem (difficulty/IQ "–" – fond je moc velký)
	if len(guesses) > 0 {
		_, _, full := progress.WordsLeft(true)
		g0 := dict.StripDiacritic(guesses[0])
		gw := &dict.DictionaryWord{Word: g0, WithoutDiacritics: g0}
		_, _, ls := calcOdds(gw, full, progress)
		luckMap = map[string]*LuckStat{g0: ls}
	}

	for _, raw := range guesses {
		guess := dict.StripDiacritic(raw)
		if len(guess) == 0 {
			continue
		}
		progress.ResetRound()
		progress.Guess(guess, solution)
		counter, counterNotUsed, wordsLeft := progress.WordsLeft(true)
		if counter == 1 && len(wordsLeft) > 0 && wordsLeft[0].WithoutDiacritics == guess {
			counter, counterNotUsed = 0, 0 // tip byl řešení
		}

		row := Row{Word: strings.ToUpper(guess), Left: counter, LeftAnswers: counterNotUsed,
			Difficulty: -1, IQ: -1, Luck: -1}
		if luckMap != nil {
			row.Luck = luckPct(luckMap[guess], counterNotUsed)
		}
		if skillMap != nil {
			if sk, ok := skillMap[guess]; ok && sk.Difficulty > 0 {
				row.Difficulty = sk.Difficulty
				row.IQ = sk.Relative
			}
		}
		rows = append(rows, row)
		if counter == 0 {
			break
		}

		// metriky pro PŘÍŠTÍ tip nad zbývajícími slovy (když fond není moc velký)
		if counter < oddsThreshold {
			weighted := make([]odds.WeightedWord, len(wordsLeft))
			newLuck := make(map[string]*LuckStat, len(wordsLeft))
			for i, dw := range wordsLeft {
				human, _, ls := calcOdds(dw, wordsLeft, progress)
				weighted[i] = odds.WeightedWord{Word: dw.Word, Weight: human}
				newLuck[dw.WithoutDiacritics] = ls
			}
			sort.Sort(odds.ByWeight(weighted))
			skillMap = odds.CalculateSkill(weighted)
			luckMap = newLuck
		} else {
			luckMap, skillMap = nil, nil
		}
	}
	return rows
}
