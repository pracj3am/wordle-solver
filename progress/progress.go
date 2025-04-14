package progress

import (
	"maps"

	"../dict"
)

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

// variable písmeno je s háčkama
// variable letter je bez háčků

type LetterFreq struct {
	f     int
	exact bool
	floor bool
}

type PositionProgress struct {
	solved  bool
	písmeno rune
	left    map[rune]bool
}

func (pp *PositionProgress) valid(index int) bool {
	if pp.solved && pp.písmeno == dict.Písmena[index] {
		return true
	}

	if !pp.solved && pp.left[dict.Letters[index]] {
		return true
	}
	return false
}

type Progress struct {
	words *dict.Dictionary
	pos   []PositionProgress
	freq  map[rune]*LetterFreq // frequency of letters for unsolved positions
}

func NewProgress(size int, words *dict.Dictionary) *Progress {
	var progress = Progress{words: words}
	progress.pos = make([]PositionProgress, size)
	for i := 0; i < size; i++ {
		progress.pos[i].left = make(map[rune]bool)
		for _, l := range letters {
			progress.pos[i].left[l] = true
		}
	}

	return &progress
}

func (p *Progress) Clone() *Progress {
	var p1 = Progress{words: p.words}
	p1.freq = p.freq
	p1.pos = make([]PositionProgress, len(p.pos))
	for i := range p.pos {
		p1.pos[i] = p.pos[i]
		if !p.pos[i].solved {
			p1.pos[i].left = maps.Clone(p.pos[i].left)
		} else {
			p1.pos[i].left = nil
		}
	}

	return &p1
}

func (p *Progress) ResetRound() {
	for l, f := range p.freq {
		if f.exact && f.f == 0 {
			for i := range p.pos {
				if p.pos[i].left != nil {
					p.pos[i].left[l] = false
				}
			}
		}
	}
	p.freq = make(map[rune]*LetterFreq)
}

func (p *Progress) incFreq(letter rune) {
	if p.freq[letter] == nil {
		p.freq[letter] = new(LetterFreq)
	}
	p.freq[letter].f++
}

func (p *Progress) Grey(i int, letter rune) {
	if p.freq[letter] == nil {
		p.freq[letter] = new(LetterFreq)
	}
	p.freq[letter].exact = true
	p.pos[i].left[letter] = false
}

func (p *Progress) Orange(i int, letter rune) {
	p.incFreq(letter)
	p.freq[letter].floor = true
	p.pos[i].left[letter] = false
}

func (p *Progress) GreenOrange(i int, písmeno rune) {
	letter := dict.Conv[písmeno]
	p.incFreq(letter)
	p.freq[letter].floor = true
	p.Green(i, písmeno)
}

func (p *Progress) Green(i int, písmeno rune) {
	letter := dict.Conv[písmeno]
	p.incFreq(letter)
	p.pos[i].solved = true
	p.pos[i].písmeno = písmeno
}

func (p *Progress) Guess(word, solution string) {
	solPísm := make([]rune, 5)
	solLtrs := make([]rune, 5)
	solLtrsPos := make(map[rune][]int, 5)
	i := 0
	for _, písmeno := range solution {
		r := dict.Conv[písmeno]
		solPísm[i] = písmeno
		solLtrs[i] = r
		solLtrsPos[r] = append(solLtrsPos[r], i)
		i++
	}

	i = 0
	for _, písmeno := range word {
		r := dict.Conv[písmeno]
		if r == solLtrs[i] { // uhodnul jsem písmeno na pozici i
			p.Green(i, solPísm[i])
		} else { // písmeno r na pozici i je špatně
			p.pos[i].left[r] = false
		}
		i++
	}
	// zjistime, jestli pismeno nemá být oranžové
	for _, písmeno := range word {
		r := dict.Conv[písmeno]
		var (
			i, j     int
			oranzova bool
		)

		for j, i = range solLtrsPos[r] {
			if !p.pos[i].solved {
				oranzova = true
				break
			}
		}

		// vyhodíme písmenko použité pro oranžovou, aby se nepoužilo víckrát
		if j+1 < len(solLtrsPos[r]) {
			solLtrsPos[r] = solLtrsPos[r][j+1:]
		} else {
			solLtrsPos[r] = nil
		}

		if oranzova {
			p.incFreq(r)
			p.freq[r].floor = true
		} else { // pismenko r se na jine pozici ve slove nevyskytuje
			if p.freq[r] == nil {
				p.freq[r] = new(LetterFreq)
			}
			p.freq[r].exact = true
		}
	}

}

func freq(l rune, indexes []int) int {
	freq := 0
	for _, idx := range indexes {
		if l == dict.Letters[idx] {
			freq++
		}
	}
	return freq
}

func (p *Progress) valid(indexes ...int) bool {
	for l, f := range p.freq {
		if f.exact || !f.floor {
			if f.f != freq(l, indexes) {
				return false
			}
		} else { // f.floor
			if f.f > freq(l, indexes) {
				return false
			}
		}
	}

	return true
}

func (p *Progress) WordsLeft(list bool) (int, int, []string) {
	counter := 0
	counterNotUsed := 0
	wordsLeft := make([]string, 0)

	for l1, w1 := range p.words.First {
		if w1 == nil {
			continue
		}
		if p.pos[0].valid(l1) {
			for l2, w2 := range w1.Next {
				if w2 == nil {
					continue
				}
				if p.pos[1].valid(l2) {
					for l3, w3 := range w2.Next {
						if w3 == nil {
							continue
						}
						if p.pos[2].valid(l3) {
							for l4, w4 := range w3.Next {
								if w4 == nil {
									continue
								}
								if p.pos[3].valid(l4) {
									for l5, w5 := range w4.Next {
										if w5 == nil {
											continue
										}
										if p.pos[4].valid(l5) {
											if p.valid(l1, l2, l3, l4, l5) {
												counter++
												word := w5.Word
												if !word.Used {
													counterNotUsed++
												}
												if list {
													wordsLeft = append(wordsLeft, word.Word)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return counter, counterNotUsed, wordsLeft
}
