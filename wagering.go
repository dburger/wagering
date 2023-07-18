package wagering

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

func average(odds ...Odds) Odds {
	sum := 0.0
	for _, odd := range odds {
		sum += odd.decimalOdds
	}
	avg := sum / float64(len(odds))
	return NewOddsFromDecimal(avg)
}

func (odds Odds) impliedProb() Probability {
	return NewProbabilityFromDecimal(1 / odds.decimalOdds)
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
