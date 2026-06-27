// Package analyzer poskytuje analýzu odehrané hry (Wordle) jako knihovnu:
// pro každý tip spočítá zbývající slova, obtížnost, "IQ" a štěstí.
// Metriky se počítají živě; pro 1. tah se použije předpočítaný luck.gob
// (GenerateLuck → NewEngine s luckPath), bez něj má 1. tah difficulty/IQ "–".
package analyzer

import (
	"encoding/gob"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/pracj3am/wordle-solver/dict"
	"github.com/pracj3am/wordle-solver/odds"
	pr "github.com/pracj3am/wordle-solver/progress"
)

// defaultOddsThreshold = výchozí Engine.OddsThreshold (viz tam).
const defaultOddsThreshold = 150

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
	Answers     []string `json:"answers"`    // zbývající možné odpovědi (cap, s diakritikou)
	Others      []string `json:"others"`     // ostatní zbývající platná slova (cap)
}

// wordsCap = max. počet slov v každém seznamu (zbytek se zkrátí, frontend ukáže „…+N").
const wordsCap = 200

// Engine drží načtený slovník. Možné odpovědi (answers) mají Used=false,
// ostatní platná slova Used=true (nejsou možné odpovědi).
// luck/skill* jsou předpočítané hodnoty pro 1. tah (z luck.gob), nebo nil.
type Engine struct {
	dict       *dict.Dictionary
	luck       map[string]*LuckStat
	skillRobot map[string]*odds.Skill
	skillHuman map[string]*odds.Skill

	// OddsThreshold: nad tolik kandidátů se obtížnost/IQ/luck (pro DALŠÍ tah) nepočítá
	// živě — výpočet je ~O(N³) (calcOdds pro každé zbylé slovo), takže pro velký fond
	// trvá na pomalém CPU desítky sekund. 1. tah má metriky z luck.gob, takže nevadí,
	// že větší fondy vyjdou jako "–". Nastavitelné zvenčí (NewEngine dá default).
	OddsThreshold int
}

// loadDict načte slovník s Used=true pro slova, která NEJSOU možná odpověď.
func loadDict(dictPath string, answers []string) (*dict.Dictionary, error) {
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
	return dict.LoadDictionary(dictPath, history)
}

// NewEngine načte slovník; je-li luckPath != "" a soubor existuje, načte i
// předpočítané luck.gob (pro 1. tah). Selhání načtení gobu není fatální.
func NewEngine(dictPath, luckPath string, answers []string) (*Engine, error) {
	d, err := loadDict(dictPath, answers)
	if err != nil {
		return nil, err
	}
	e := &Engine{dict: d, OddsThreshold: defaultOddsThreshold}
	if luckPath != "" {
		if luck, sr, sh, err := LoadLuck(luckPath); err == nil {
			e.luck, e.skillRobot, e.skillHuman = luck, sr, sh
		}
	}
	return e, nil
}

func LoadLuck(path string) (map[string]*LuckStat, map[string]*odds.Skill, map[string]*odds.Skill, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var luck map[string]*LuckStat
	var sr, sh map[string]*odds.Skill
	if err := dec.Decode(&luck); err != nil {
		return nil, nil, nil, err
	}
	if err := dec.Decode(&sr); err != nil {
		return nil, nil, nil, err
	}
	if err := dec.Decode(&sh); err != nil {
		return nil, nil, nil, err
	}
	return luck, sr, sh, nil
}

// GenerateLuck předpočítá luck.gob pro 1. tah: pro každé slovo fondu spočítá
// (paralelně) calcOdds nad celým fondem → histogram štěstí + váhy, z vah skill.
func GenerateLuck(dictPath string, answers []string, outPath string) error {
	d, err := loadDict(dictPath, answers)
	if err != nil {
		return err
	}
	base := pr.NewProgress(5, d)
	_, _, all := base.WordsLeft(true) // všechna slova (bez omezení)

	type result struct {
		word         string
		human, robot float64
		luck         *LuckStat
	}
	results := make([]result, len(all))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, dw := range all {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, dw *dict.DictionaryWord) {
			defer wg.Done()
			defer func() { <-sem }()
			h, r, l := calcOdds(dw, all, base)
			results[i] = result{dw.Word, h, r, l}
		}(i, dw)
	}
	wg.Wait()

	luck := make(map[string]*LuckStat, len(all))
	robotW := make([]odds.WeightedWord, len(all))
	humanW := make([]odds.WeightedWord, len(all))
	for i, r := range results {
		luck[dict.StripDiacritic(r.word)] = r.luck
		robotW[i] = odds.WeightedWord{Word: r.word, Weight: r.robot}
		humanW[i] = odds.WeightedWord{Word: r.word, Weight: r.human}
	}
	sort.Sort(odds.ByWeight(robotW))
	skillRobot := odds.CalculateSkill(robotW)
	sort.Sort(odds.ByWeight(humanW))
	skillHuman := odds.CalculateSkill(humanW)

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	if err := enc.Encode(luck); err != nil {
		return err
	}
	if err := enc.Encode(skillRobot); err != nil {
		return err
	}
	return enc.Encode(skillHuman)
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
	luckMap := e.luck        // z luck.gob (pokrývá 1. tah – luck i difficulty/IQ), nebo nil
	skillMap := e.skillHuman

	// bez gobu: fallback – štěstí 1. tahu nad plným fondem (difficulty/IQ zůstane "–")
	if luckMap == nil && len(guesses) > 0 {
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
		// seznamy zbývajících slov (možné odpovědi vs ostatní platná), dedup + cap
		if counter > 0 {
			seen := make(map[string]bool, len(wordsLeft))
			for _, dw := range wordsLeft {
				if seen[dw.WithoutDiacritics] {
					continue
				}
				seen[dw.WithoutDiacritics] = true
				if dw.Used { // ostatní platná (není možná odpověď)
					if len(row.Others) < wordsCap {
						row.Others = append(row.Others, dw.Word)
					}
				} else { // možná odpověď
					if len(row.Answers) < wordsCap {
						row.Answers = append(row.Answers, dw.Word)
					}
				}
			}
			sort.Strings(row.Answers)
			sort.Strings(row.Others)
		}
		rows = append(rows, row)
		if counter == 0 {
			break
		}

		// metriky pro PŘÍŠTÍ tip nad zbývajícími slovy (když fond není moc velký)
		if counter < e.OddsThreshold {
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
