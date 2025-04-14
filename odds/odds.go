package odds

import (
	"../dict"
)

type Skill struct {
	Relative   int
	Difficulty int
}

type WeightedWord struct {
	Word   string
	Weight float64
}

type ByWeight []WeightedWord

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].Weight < a[j].Weight }

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
		w := dict.StripDiacritic(s.w)
		skill[w] = &Skill{Relative: sk, Difficulty: maxSk}
	}

	return skill
}
