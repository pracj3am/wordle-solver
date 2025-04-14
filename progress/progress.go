package progress

import (
	"slices"

	"../dict"
)

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
	left    []bool // indexed by letter index
}

func (pp *PositionProgress) valid(index int) bool {
	if pp.solved && pp.písmeno == dict.Písmena[index] {
		return true
	}

	if !pp.solved && pp.left[dict.ConvIndex[index]] {
		return true
	}

	return false
}

type Progress struct {
	words *dict.Dictionary
	pos   []PositionProgress
	freq  []LetterFreq // index by letter index: frequency of letters for unsolved positions
}

func NewProgress(size int, words *dict.Dictionary) *Progress {
	var progress = Progress{words: words}
	progress.pos = make([]PositionProgress, size)
	progress.freq = make([]LetterFreq, 26)
	for i := 0; i < size; i++ {
		progress.pos[i].left = make([]bool, 26)
		for j := range progress.pos[i].left {
			progress.pos[i].left[j] = true
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
			p1.pos[i].left = slices.Clone(p.pos[i].left)
		} else {
			p1.pos[i].left = nil
		}
	}

	return &p1
}

func (p *Progress) ResetRound() {
	for j, f := range p.freq {
		if f.exact && f.f == 0 {
			for i := range p.pos {
				if p.pos[i].left != nil {
					p.pos[i].left[j] = false
				}
			}
		}
	}

	p.freq = make([]LetterFreq, 26)
}

func (p *Progress) Grey(i int, letter rune) {
	index := dict.Indexes[letter]
	p.freq[index].exact = true
	p.pos[i].left[index] = false
}

func (p *Progress) Orange(i int, letter rune) {
	index := dict.Indexes[letter]
	p.freq[index].f++
	p.freq[index].floor = true
	p.pos[i].left[index] = false
}

func (p *Progress) GreenOrange(i int, písmeno rune) {
	index := dict.ConvIndex[dict.Indexes[písmeno]]
	p.freq[index].f++
	p.freq[index].floor = true
	p.Green(i, písmeno)
}

func (p *Progress) Green(i int, písmeno rune) {
	index := dict.ConvIndex[dict.Indexes[písmeno]]
	p.freq[index].f++
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
			p.pos[i].left[dict.ConvIndex[dict.Indexes[písmeno]]] = false
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

		index := dict.ConvIndex[dict.Indexes[písmeno]]
		if oranzova {
			p.freq[index].f++
			p.freq[index].floor = true
		} else { // pismenko r se na jine pozici ve slove nevyskytuje
			p.freq[index].exact = true
		}
	}

}

func freq(i int, indexes []int) int {
	freq := 0
	for _, idx := range indexes {
		if i == dict.ConvIndex[idx] {
			freq++
		}
	}
	return freq
}

func (p *Progress) valid(indexes ...int) bool {
	for i, f := range p.freq {
		if f.exact || (!f.floor && f.f > 0) {
			if f.f != freq(i, indexes) {
				return false
			}
		} else if f.floor {
			if f.f > freq(i, indexes) {
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
