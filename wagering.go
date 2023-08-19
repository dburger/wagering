package wagering

import "math"

type Odds struct {
	decimalOdds  float64
	americanOdds float64
}

func NewOddsFromAmerican(americanOdds float64) Odds {
	var decimalOdds float64
	if americanOdds > 0 {
		decimalOdds = americanOdds/100.0 + 1.0
	} else {
		decimalOdds = 1.0 - 100.0/americanOdds
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds}
}

func NewOddsFromDecimal(decimalOdds float64) Odds {
	var americanOdds float64
	if decimalOdds >= 2.0 {
		americanOdds = (decimalOdds - 1.0) * 100.0
	} else {
		americanOdds = -100.0 / (decimalOdds - 1.0)
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds}
}

type AverageOdds struct {
	sum   float64
	count int
}

func NewAverageOdds() AverageOdds {
	return AverageOdds{}
}

func (ao *AverageOdds) Accumulate(odds ...Odds) {
	for _, o := range odds {
		ao.sum += o.decimalOdds
		ao.count++
	}
}

func (ao *AverageOdds) Average() Odds {
	return NewOddsFromDecimal(ao.sum / float64(ao.count))
}

func (ao *AverageOdds) AverageWithout(odds Odds, count int) Odds {
	sum := ao.sum - (odds.decimalOdds * float64(count))
	decimalOdds := sum / float64(ao.count-count)
	return NewOddsFromDecimal(decimalOdds)
}

func TrueOddsNormalized(odds ...Odds) []Odds {
	probs := []Probability{}
	for _, o := range odds {
		probs = append(probs, o.ImpliedProb())
	}
	probSum := 0.0
	for _, p := range probs {
		probSum += p.decimal
	}
	norms := []Odds{}
	for _, o := range odds {
		norms = append(norms, NewOddsFromDecimal(o.decimalOdds*probSum))
	}
	return norms
}

func (odds Odds) American() float64 {
	return odds.americanOdds
}

func (odds Odds) Decimal() float64 {
	return odds.decimalOdds
}

func (odds Odds) KellyPercent(prob Probability, mult float64) float64 {
	profitMult := odds.decimalOdds - 1.0
	kelly := (profitMult*prob.decimal - (1.00 - prob.decimal)) / profitMult
	percent := mult * kelly
	return math.Max(percent, 0.0)
}

func (odds Odds) KellyStake(prob Probability, mult, stake float64) float64 {
	return odds.KellyPercent(prob, mult) * stake
}

func (odds Odds) Equals(other Odds) bool {
	return odds.decimalOdds == other.decimalOdds
}

func (odds Odds) Longer(other Odds) bool {
	return odds.decimalOdds > other.decimalOdds
}

func (odds Odds) Shorter(other Odds) bool {
	return odds.decimalOdds < other.decimalOdds
}

func (odds Odds) ImpliedProb() Probability {
	return NewProbabilityFromDecimal(1 / odds.decimalOdds)
}

func (odds Odds) ExpectedValuePercent(prob Probability) float64 {
	return prob.decimal*(odds.decimalOdds-1.0) - (1.0 - prob.decimal)
}

func MarketWidth(odds1, odds2 Odds) float64 {
	if odds1.americanOdds < 0 && odds2.americanOdds < 0 {
		return math.Abs(odds1.americanOdds) + math.Abs(odds2.americanOdds) - 200.0
	} else if odds1.americanOdds > 0 && odds2.americanOdds > 0 {
		// My own concoction, both positive becomes a negative market width.
		return -(odds1.americanOdds + odds2.americanOdds - 200.0)
	} else {
		return math.Abs(math.Abs(odds1.americanOdds) - math.Abs(odds2.americanOdds))
	}
}

type Probability struct {
	decimal float64
	percent float64
}

func NewProbabilityFromPercent(percent float64) Probability {
	return Probability{percent / 100.0, percent}
}

func NewProbabilityFromDecimal(decimal float64) Probability {
	return Probability{decimal, decimal * 100.0}
}
