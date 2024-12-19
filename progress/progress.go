package progress

import (
	"maps"

	"../dict"
)

type LetterFreq struct {
	f     int
	exact bool
	floor bool
}

type PositionProgress struct {
	solved bool
	letter rune
	left   map[rune]bool
}

func (pp *PositionProgress) Valid(r rune) bool {
	if pp.solved && pp.letter == r {
		return true
	}
	if !pp.solved && pp.left[r] {
		return true
	}
	return false
}

type Progress struct {
	pos  []PositionProgress
	freq map[rune]*LetterFreq // frequency of letters for unsolved positions
}

func NewProgress(size int, letters []rune) *Progress {
	var progress Progress
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
	var p1 Progress
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

func (p *Progress) IncFreq(letter rune) {
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
	p.IncFreq(letter)
	p.freq[letter].floor = true
	p.pos[i].left[letter] = false
}

func (p *Progress) GreenOrange(i int, letter rune) {
	p.IncFreq(letter)
	p.freq[letter].floor = true
	p.Green(i, letter)
}

func (p *Progress) Green(i int, letter rune) {
	p.IncFreq(letter)
	p.pos[i].solved = true
	p.pos[i].letter = letter
}

func (p *Progress) Guess(word, solution string) {
	solLtrs := make([]rune, 5)
	solLtrsPos := make(map[rune][]int, 5)
	for i, r := range solution {
		solLtrs[i] = r
		solLtrsPos[r] = append(solLtrsPos[r], i)
	}

	for i, r := range word {
		if r == solLtrs[i] { // uhodnul jsem písmeno na pozici i
			p.Green(i, r)
		} else { // písmeno r na pozici i je špatně
			p.pos[i].left[r] = false
		}
	}
	// zjistime, jestli pismeno nemá být oranžové
	for _, r := range word {
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
			p.IncFreq(r)
			p.freq[r].floor = true
		} else { // pismenko r se na jine pozici ve slove nevyskytuje
			if p.freq[r] == nil {
				p.freq[r] = new(LetterFreq)
			}
			p.freq[r].exact = true
		}
	}

}

func (p *Progress) Valid(letters ...rune) bool {
	freq := make(map[rune]int)
	for _, r := range letters {
		freq[r]++
	}
	for l, f := range p.freq {
		if f.exact || !f.floor {
			if f.f != freq[l] {
				return false
			}
		} else { // f.floor
			if f.f > freq[l] {
				return false
			}
		}
	}

	return true
}

func (p *Progress) WordsLeft(words dict.Dictionary, list bool) (int, int, []string) {
	counter := 0
	counterNotUsed := 0
	wordsLeft := make([]string, 0)

	for l1, w1 := range words {
		if p.pos[0].Valid(l1) {
			for l2, w2 := range w1 {
				if p.pos[1].Valid(l2) {
					for l3, w3 := range w2 {
						if p.pos[2].Valid(l3) {
							for l4, w4 := range w3 {
								if p.pos[3].Valid(l4) {
									for l5, w5 := range w4 {
										if p.pos[4].Valid(l5) {
											if p.Valid(l1, l2, l3, l4, l5) {
												counter++
												if !w5.Used {
													counterNotUsed++
												}
												if list {
													wordsLeft = append(wordsLeft, w5.Word)
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
