/*
Package wagering provides types for representing odds and performing computations
related to wagering.

Typical usage:

	likelihood := 0.6
	multiplier := 0.3
	bankroll := 10000.0
	odds := wagering.NewOddsFromAmerican(-110.0)
	wager := odds.KellyStake(likelihood, multiplier, bankroll)

Note that when odds are constructed from american or decimal odds, that value is
held explicitly and the other format is computed but may suffer from minor rounding
skew.
*/
package wagering

import (
	"math"
)

type Odds struct {
	decimalOdds  float64
	americanOdds float64
}

// NewOddsFromAmerican constructs a new Odds from the given american odds.
func NewOddsFromAmerican(americanOdds float64) Odds {
	var decimalOdds float64
	if americanOdds > 0 {
		decimalOdds = americanOdds/100.0 + 1.0
	} else {
		decimalOdds = 1.0 - 100.0/americanOdds
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds}
}

// NewOddsFromDecimal constructs a new Odds from the given decimal odds.
func NewOddsFromDecimal(decimalOdds float64) Odds {
	var americanOdds float64
	if decimalOdds >= 2.0 {
		americanOdds = (decimalOdds - 1.0) * 100.0
	} else {
		americanOdds = -100.0 / (decimalOdds - 1.0)
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds}
}

// AverageOdds provides a way to compute the average of a number of Odds.
type AverageOdds struct {
	sum   float64
	count int
}

// NewAverageOdds construts a new AverageOdds.
func NewAverageOdds() AverageOdds {
	return AverageOdds{}
}

// Accumulate accumulates Odds into AverageOdds.
func (ao *AverageOdds) Accumulate(odds ...Odds) {
	for _, o := range odds {
		ao.sum += o.decimalOdds
		ao.count++
	}
}

// Average returns the average Odds for the AverageOdds.
func (ao *AverageOdds) Average() Odds {
	return NewOddsFromDecimal(ao.sum / float64(ao.count))
}

// AverageWithout returns the Odds for AverageOdds with a count of Odds removed.
// This most obvious usage of this is to give the average odds while disregarding
// a particular value that was already accumulated into AverageOdds.
func (ao *AverageOdds) AverageWithout(odds Odds, count int) Odds {
	sum := ao.sum - (odds.decimalOdds * float64(count))
	decimalOdds := sum / float64(ao.count-count)
	return NewOddsFromDecimal(decimalOdds)
}

func probs(odds ...Odds) []Probability {
	var probs []Probability
	for _, o := range odds {
		probs = append(probs, o.ImpliedProb())
	}
	return probs
}

// probSum returns the summation of the implied probabilities for the given odds.
func probSum(odds ...Odds) float64 {
	probs := probs(odds...)
	probSum := 0.0
	for _, p := range probs {
		probSum += p.decimal
	}
	return probSum
}

func margin(odds ...Odds) float64 {
	return probSum(odds...) - 1.0
}

// American returns the american odds.
func (odds Odds) American() float64 {
	return odds.americanOdds
}

// Decimal returns the decimal odds.
func (odds Odds) Decimal() float64 {
	return odds.decimalOdds
}

// KellyFraction returns the fraction of the bankroll to wager given the probability
// for success and kelly multiplier.
// https://en.wikipedia.org/wiki/Kelly_criterion
func (odds Odds) KellyFraction(prob Probability, mult float64) float64 {
	profitMult := odds.decimalOdds - 1.0
	kelly := (profitMult*prob.decimal - (1.00 - prob.decimal)) / profitMult
	percent := mult * kelly
	return math.Max(percent, 0.0)
}

// KellyStake returns the amount that should be wagered given the probability of success,
// kelly multiplier, and total bankroll.
// https://en.wikipedia.org/wiki/Kelly_criterion
func (odds Odds) KellyStake(prob Probability, mult, bankroll float64) float64 {
	return odds.KellyFraction(prob, mult) * bankroll
}

// Equals returns whether odds is equal to the given odds.
func (odds Odds) Equals(other Odds) bool {
	return odds.decimalOdds == other.decimalOdds
}

// Longer returns whether odds is longer than the given odds.
func (odds Odds) Longer(other Odds) bool {
	return odds.decimalOdds > other.decimalOdds
}

// Shorter returns whether odds is shorter than the given odds.
func (odds Odds) Shorter(other Odds) bool {
	return odds.decimalOdds < other.decimalOdds
}

// ImpliedProb returns the implied probability of the given odds.
// This computation is equivalent to the break even probability.
func (odds Odds) ImpliedProb() Probability {
	return NewProbabilityFromDecimal(1 / odds.decimalOdds)
}

// ExpectedValueProb returns the long term expected value when wagering odds
// at the given probability. The result is given as the percent increase or
// decrease (negative) of the wager.
func (odds Odds) ExpectedValueProb(prob Probability) float64 {
	return prob.decimal*(odds.decimalOdds-1.0) - (1.0 - prob.decimal)
}

// ExpectedValueOdds returns the long term expected value when wagering odds
// at the given true odds.  The result is given as the percent increase or
// decrease (negative) of the wager.
func (odds Odds) ExpectedValueOdds(trueOdds Odds) float64 {
	return odds.ExpectedValueProb(trueOdds.ImpliedProb())
}

// MarketWidth returns the market width between the given odds.
func MarketWidth(odds1, odds2 Odds) float64 {
	if odds1.americanOdds < 0 && odds2.americanOdds < 0 {
		return math.Abs(odds1.americanOdds) + math.Abs(odds2.americanOdds) - 200.0
	} else if odds1.americanOdds > 0 && odds2.americanOdds > 0 {
		// My own concoction, both positive becomes a negative market width.
		return -(odds1.americanOdds + odds2.americanOdds - 200.0)
	} else {
		return math.Abs(odds1.americanOdds + odds2.americanOdds)
	}
}

// Probability represents a probability and stores the decimal and percent
// representations. By using Probability, instead of a float, the ambiguity
// between passing the decimal or percent is removed.
type Probability struct {
	decimal float64
	percent float64
}

// NewProbabilityFromPercent constructs a Probability from the given percent.
func NewProbabilityFromPercent(percent float64) Probability {
	return Probability{percent / 100.0, percent}
}

// NewProbabilityFromDecimal constructs a Probability from the given decimal.
func NewProbabilityFromDecimal(decimal float64) Probability {
	return Probability{decimal, decimal * 100.0}
}

// Pro bettor nishikori says:
// in Football, the methods that seem to come closest to the true odds are
// "Margin proportional to odds" and "Logarithmic", whereas in Tennis are
// the "Odds ratio" and ""Margin proportional to odds".

// For further reading on these algorithms to determine "true odds" see the
// following resources:
// https://www.football-data.co.uk/The_Wisdom_of_the_Crowd_updated.pdf
// https://outlier.bet/wp-content/uploads/2023/08/2017-clarke-adjusting_bookmakers_odds.pdf
// https://winnerodds.com/valuebettingblog/true-odds-calculator/

// EqualMarginOdds gives the odds of the given Odds using the method of simple normalization.
func EqualMarginOdds(odds ...Odds) []Odds {
	probSum := probSum(odds...)
	var norms []Odds
	for _, o := range odds {
		norms = append(norms, NewOddsFromDecimal(o.decimalOdds*probSum))
	}
	return norms
}

// AdditiveOdds gives the odds of the given Odds by removing equal amounts of the margin.
func AdditiveOdds(odds ...Odds) []Odds {
	n := float64(len(odds))
	m := margin(odds...)
	var norms []Odds
	for _, o := range odds {
		prob := 1/o.decimalOdds - m/n
		norms = append(norms, NewOddsFromDecimal(1/prob))
	}
	return norms
}

// MPTOdds implements the "margin proportional to odds" approach.
func MPTOdds(odds ...Odds) []Odds {
	n := float64(len(odds))
	m := margin(odds...)
	var norms []Odds
	for _, o := range odds {
		norms = append(norms, NewOddsFromDecimal((n*o.decimalOdds)/(n-m*o.decimalOdds)))
	}
	return norms
}

// Largely taken from https://github.com/mberk/shin.
func ShinOdds(odds ...Odds) []Odds {
	// TODO(dburger): if n == 2 use additive
	n := len(odds)
	probs := probs(odds...)
	overround := probSum(odds...)
	delta := math.MaxFloat64
	convergenceThreshold := 1e-12
	z := 0.0
	iterations := 0
	maxIterations := 1000

	for delta > convergenceThreshold && iterations < maxIterations {
		z0 := z
		z = -2.0
		for _, p := range probs {
			z += math.Sqrt(math.Pow(z0, 2) + 4*(1-z0)*math.Pow(p.decimal, 2)/overround)
		}

		// why isn't this division by zero on a two banger?
		z /= (float64(n) - 2.0)
		delta = math.Abs(z - z0)
		iterations++
	}

	// Now use z to make the true odds.
	var trueOdds []Odds
	for _, p := range probs {
		prob := (math.Sqrt(math.Pow(z, 2)+4*(1-z)*math.Pow(p.decimal, 2)/overround) - z) / (2 * (1 - z))
		trueOdds = append(trueOdds, NewOddsFromDecimal(1/prob))
	}
	return trueOdds
}

// https://www.sportstradingnetwork.com/article/fixed-odds-betting-traditional-odds/
func OddsRatioOdds(odds ...Odds) []Odds {
	delta := math.MaxFloat64
	convergenceThreshold := 1e-12
	diff := 0.0
	c := 1.0
	iterations := 0
	maxIterations := 1000
	for delta > convergenceThreshold && iterations < maxIterations {
		c -= diff
		sum := 0.0
		for _, o := range odds {
			sum += 1 / (c*o.decimalOdds + 1 - c)
		}
		diff = 1.0 - sum
		delta = math.Abs(diff)
		iterations++
	}

	// Now use c to make the true odds.
	var trueOdds []Odds
	for _, o := range odds {
		trueOdds = append(trueOdds, NewOddsFromDecimal(1/(1/(c*o.decimalOdds+1-c))))
	}
	return trueOdds

}

func LogarithmicOdds(odds ...Odds) []Odds {
	probs := probs(odds...)
	delta := math.MaxFloat64
	convergenceThreshold := 1e-12
	diff := 0.0
	c := 1.0
	iterations := 0
	maxIterations := 1000
	for delta > convergenceThreshold && iterations < maxIterations {
		c -= diff
		sum := 0.0
		for _, p := range probs {
			sum += math.Pow(p.decimal, c)
		}
		diff = 1.0 - sum
		delta = math.Abs(diff)
		iterations++
	}

	// Now use c to make the true odds.
	var trueOdds []Odds
	for _, p := range probs {
		trueOdds = append(trueOdds, NewOddsFromDecimal(1.0/math.Pow(p.decimal, c)))
	}
	return trueOdds
}
