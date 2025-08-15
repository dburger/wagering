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
	"fmt"
	"math"
)

type Odds struct {
	decimalOdds  float64
	americanOdds float64
}

type OddsFormat struct {
	slug string
}

// TODO(dburger): how to prevent accidental overwrite?
// Should I make these unexported and only return them from FromString?

var (
	Unknown  = OddsFormat{""}
	American = OddsFormat{"american"}
	Decimal  = OddsFormat{"decimal"}
)

func FromString(s string) (OddsFormat, error) {
	switch s {
	case American.slug:
		return American, nil
	case Decimal.slug:
		return Decimal, nil
	default:
		return Unknown, fmt.Errorf("unknown odds format: %v", s)
	}
}

func (of OddsFormat) ToString() string {
	return of.slug
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

// American returns the american odds.
func (odds Odds) American() float64 {
	return odds.americanOdds
}

// Decimal returns the decimal odds.
func (odds Odds) Decimal() float64 {
	return odds.decimalOdds
}

func (odds Odds) ToString(of OddsFormat) string {
	if of == American {
		if odds.americanOdds >= 0 {
			return fmt.Sprintf("+%.2f", odds.americanOdds)
		} else {
			return fmt.Sprintf("-%.2f", odds.americanOdds)
		}
	} else if of == Decimal {
		return fmt.Sprintf("%.2f", odds.decimalOdds)
	} else {
		panic("unknown odds format")
	}
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

func transSum(prob func(Odds) float64, odds ...Odds) float64 {
	probSum := 0.0
	for _, o := range odds {
		probSum += prob(o)
	}
	return probSum
}

func transOdds(prob func(Odds) float64, odds ...Odds) []Odds {
	var trans []Odds
	for _, o := range odds {
		trans = append(trans, NewOddsFromDecimal(1.0/prob(o)))
	}
	return trans
}

func margin(odds ...Odds) float64 {
	return probSum(odds...) - 1.0
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

func ShinOdds(odds ...Odds) []Odds {
	tolerance := 1e-12
	maxIterations := 1000
	c := 0.0
	i := 0
	overround := probSum(odds...)

	prob := func(odds Odds) float64 {
		sqrt := math.Sqrt((math.Pow(c, 2.0) + 4.0*(1.0-c)*math.Pow(odds.ImpliedProb().decimal, 2.0)) / overround)
		numerator := sqrt - c
		denominator := 2.0 * (1.0 - c)
		return numerator / denominator
	}

	probSum := transSum(prob, odds...)
	delta := probSum - 1.0

	for math.Abs(delta) > tolerance && i < maxIterations {
		c += delta
		probSum = transSum(prob, odds...)
		delta = probSum - 1.0
		i++
	}

	// Now use c to make the true odds.
	return transOdds(prob, odds...)
}

// https://www.sportstradingnetwork.com/article/fixed-odds-betting-traditional-odds/
func OddsRatioOdds(odds ...Odds) []Odds {
	tolerance := 1e-12
	maxIterations := 1000
	c := 1.0
	i := 0

	prob := func(odds Odds) float64 {
		return odds.ImpliedProb().decimal / (c + ((1.0 - c) / (odds.decimalOdds)))
	}

	probSum := transSum(prob, odds...)
	delta := probSum - 1.0

	for math.Abs(delta) > tolerance && i < maxIterations {
		c += delta
		probSum = transSum(prob, odds...)
		delta = probSum - 1.0
		i++
	}

	// Now use c to make the true odds.
	return transOdds(prob, odds...)
}

func LogarithmicOdds(odds ...Odds) []Odds {
	tolerance := 1e-12
	maxIterations := 1000
	c := 1.0
	i := 0

	prob := func(odds Odds) float64 {
		return math.Pow(1.0/odds.decimalOdds, c)
	}

	probSum := transSum(prob, odds...)
	delta := probSum - 1.0

	for math.Abs(delta) > tolerance && i < maxIterations {
		c += delta
		probSum = transSum(prob, odds...)
		delta = probSum - 1.0
		i++
	}

	// Now use c to make the true odds.
	return transOdds(prob, odds...)
}
