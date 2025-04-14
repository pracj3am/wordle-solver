package odds

import (
	"math"

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

	minW := words[0].Weight
	maxW := words[len(words)-1].Weight
	diff := 0
	diffInv := 0.0

	if minW > 0 {
		diff = int(math.Round(10 * (maxW - minW) / minW))
		diffInv = 100.0 / (maxW - minW)
	}

	for _, w := range words {
		sk := 0
		if diff > 0 {
			sk = 100 - int(math.Round((w.Weight-minW)*diffInv))
		}
		w := dict.StripDiacritic(w.Word)
		skill[w] = &Skill{Relative: sk, Difficulty: diff}
	}

	return skill
}
