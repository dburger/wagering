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
	decimalOdds   float64
	americanOdds  float64
	netFractional float64
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

func OddsFormatFromString(s string) (OddsFormat, error) {
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

// NewOdds constructs a new Odds from the given price and odds format.
func NewOdds(price float64, oddsFormat OddsFormat) (Odds, error) {
	if oddsFormat == American {
		return NewOddsFromAmerican(price), nil
	} else if oddsFormat == Decimal {
		return NewOddsFromDecimal(price), nil
	} else {
		return Odds{}, fmt.Errorf("unknown odds format: %v", oddsFormat)
	}
}

// NewOddsFromAmerican constructs a new Odds from the given american odds.
func NewOddsFromAmerican(americanOdds float64) Odds {
	var decimalOdds float64
	if americanOdds > 0 {
		decimalOdds = americanOdds/100.0 + 1.0
	} else {
		decimalOdds = 1.0 - 100.0/americanOdds
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds, netFractional: decimalOdds - 1.0}
}

// NewOddsFromDecimal constructs a new Odds from the given decimal odds.
func NewOddsFromDecimal(decimalOdds float64) Odds {
	var americanOdds float64
	if decimalOdds >= 2.0 {
		americanOdds = (decimalOdds - 1.0) * 100.0
	} else {
		americanOdds = -100.0 / (decimalOdds - 1.0)
	}
	return Odds{decimalOdds: decimalOdds, americanOdds: americanOdds, netFractional: decimalOdds - 1.0}
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
			return fmt.Sprintf("%.2f", odds.americanOdds)
		}
	} else if of == Decimal {
		return fmt.Sprintf("%.2f", odds.decimalOdds)
	} else {
		panic("unknown odds format")
	}
}

// AverageOdds provides a way to compute the average of a number of Odds. The average
// is computed as the average of the implied probabilities, weighted when Odds are given
// to AccumulateWeighted.
type AverageOdds struct {
	probSum   float64
	weightSum float64
}

// NewAverageOdds constructs a new AverageOdds.
func NewAverageOdds() AverageOdds {
	return AverageOdds{}
}

// Accumulate accumulates Odds into AverageOdds.
func (ao *AverageOdds) Accumulate(odds ...Odds) {
	ao.AccumulateWeighted(1, odds...)
}

// AccumulateWeighted accumulates Odds into AverageOdds, each counting weight times as much
// as Odds given to Accumulate.
func (ao *AverageOdds) AccumulateWeighted(weight float64, odds ...Odds) {
	ao.probSum += weight * probSum(odds...)
	ao.weightSum += weight * float64(len(odds))
}

// Average returns the average Odds for the AverageOdds.
func (ao *AverageOdds) Average() Odds {
	avgProb := ao.probSum / ao.weightSum
	// TODO(dburger): should we introduce NewOddsFromProb or NewOddsFromImpliedProb?
	return NewOddsFromDecimal(1.0 / avgProb)
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

// Meg returns the maximum expected growth, in basis points: the compounded growth
// of the bankroll per bet from betting the full Kelly fraction at odds when
// trueOdds are the real odds, (1 + f*b)^p * (1 - f)^q - 1. It is 0 without an
// edge, since Kelly then bets nothing.
// https://en.wikipedia.org/wiki/Kelly_criterion
func (odds Odds) Meg(trueOdds Odds) float64 {
	prob := trueOdds.ImpliedProb()
	fraction := odds.KellyFraction(prob, 1.0)
	p := prob.decimal
	q := 1.0 - p
	// Computed through the log growth, which Log1p and Expm1 keep accurate for
	// small fractions.
	logGrowth := p*math.Log1p(fraction*odds.netFractional) + q*math.Log1p(-fraction)
	return 10000.0 * math.Expm1(logGrowth)
}

// ArbTo returns whether the given odds is an arbitrage to other odds.
func (odds Odds) ArbTo(other Odds) bool {
	return odds.ImpliedProb().decimal+other.ImpliedProb().decimal < 1.0
	/*
		if odds.americanOdds > 0 {
			return -odds.americanOdds < other.americanOdds
		} else if other.americanOdds > 0 {
			return -other.americanOdds < odds.americanOdds
		} else {
			return false
		}
	*/
}

func (odds Odds) ArbRoi(other Odds) float64 {
	x := odds.decimalOdds
	y := other.decimalOdds
	return x*y/(x+y) - 1.0
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
func EqualMarginOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	probSum := probSum(odds...)
	var norms []Odds
	for _, o := range odds {
		norms = append(norms, NewOddsFromDecimal(o.decimalOdds*probSum))
	}
	return norms, nil
}

// MPTOdds implements the "margin proportional to odds" approach, also known as the
// additive method: it removes an equal share of the margin, m/n, from each implied
// probability.
//
// A long enough price in a wide enough book has more of the margin taken than its
// implied probability holds, which leaves it at zero or below. The others still
// sum to one with it, so none of them is fair either, and an error is returned.
func MPTOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	n := float64(len(odds))
	m := margin(odds...)
	var norms []Odds
	for _, o := range odds {
		prob := 1/o.decimalOdds - m/n
		if prob <= 0.0 {
			return nil, fmt.Errorf("margin %v exceeds the implied probability of decimal odds %v", m, o.decimalOdds)
		}
		norms = append(norms, NewOddsFromDecimal(1/prob))
	}
	return norms, nil
}

// solveForC solves for the parameter c that removes the vig, where the true
// probabilities of the odds sum to one, and returns the true odds at that c.
//
// Each method's probabilities sum to less as c grows, over the whole of the
// range c can take, and from above one at the bottom of that range to below one
// at the top. So there is exactly one answer, and bisection finds it: step out
// from start until the sum crosses one, then halve the gap until it closes. That
// cannot fail to converge.
//
// It replaced a fixed point iteration, c += sum - 1, which moved c by the whole
// overshoot each step. Where the sum is steep in c that overshoots the answer
// and swings about it forever, and the loop then gave up at its cap and returned
// wherever it was, without an error.
//
// lowest may be math.Inf(-1) and highest math.Inf(1). An error is returned when
// no c makes the probabilities sum to one, as for a price of 1.0.
func solveForC(odds []Odds, prob func(o Odds, c float64) float64, start, lowest, highest float64) ([]Odds, error) {
	excess := func(c float64) float64 {
		sum := 0.0
		for _, o := range odds {
			sum += prob(o, c)
		}
		return sum - 1.0
	}

	// outward returns the next point out from c towards limit: halfway to it if
	// it is finite, since the limit itself may not be evaluable, and a doubling
	// step if it is not.
	outward := func(c, limit, step float64) float64 {
		if math.IsInf(limit, 0) {
			return c + step
		}
		return (c + limit) / 2.0
	}

	// Bracket the answer between a c whose sum is over one and one whose sum is
	// under it.
	over, under := start, start
	atStart := excess(start)
	found := atStart == 0.0
	for i, step := 0, 1.0; i < 2000 && !found; i, step = i+1, step*2.0 {
		if atStart > 0.0 {
			under = outward(under, highest, step)
			found = excess(under) < 0.0
			if !found {
				over = under
			}
		} else {
			over = outward(over, lowest, -step)
			found = excess(over) > 0.0
			if !found {
				under = over
			}
		}
	}
	if !found {
		var decimals []float64
		for _, o := range odds {
			decimals = append(decimals, o.decimalOdds)
		}
		return nil, fmt.Errorf("no true odds for decimal odds %v", decimals)
	}

	// Halve the bracket until no float64 lies strictly inside it.
	for i := 0; i < 2000; i++ {
		mid := (over + under) / 2.0
		if mid == over || mid == under {
			break
		}
		if excess(mid) > 0.0 {
			over = mid
		} else {
			under = mid
		}
	}
	c := (over + under) / 2.0
	return transOdds(func(o Odds) float64 { return prob(o, c) }, odds...), nil
}

// ShinOdds implements Shin's approach, which models the vig as the book's
// protection against insider money. c is z, the share of the money that is
// insiders', which runs up to but not including one. It is negative for a book
// whose implied probabilities sum to less than one, which Shin's model does not
// describe but the formula still devigs.
func ShinOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	overround := probSum(odds...)

	return solveForC(odds, func(o Odds, c float64) float64 {
		// Only the pi^2 term is divided by the overround, as in Jullien and Salanie
		// (1994) and Strumbelj (2014).
		sqrt := math.Sqrt(math.Pow(c, 2.0) + 4.0*(1.0-c)*math.Pow(o.ImpliedProb().decimal, 2.0)/overround)
		numerator := sqrt - c
		denominator := 2.0 * (1.0 - c)
		return numerator / denominator
	}, 0.0, math.Inf(-1), 1.0)
}

// OddsRatioOdds implements the "odds ratio" approach, which divides the odds (in
// the p / (1 - p) sense) of every implied probability by the same c.
// https://www.sportstradingnetwork.com/article/fixed-odds-betting-traditional-odds/
func OddsRatioOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	// c runs over the positive numbers. Solved for in its log, since c ranges over
	// orders of magnitude and a log spreads them evenly for bisection.
	return solveForC(odds, func(o Odds, logC float64) float64 {
		c := math.Exp(logC)
		return o.ImpliedProb().decimal / (c + ((1.0 - c) / (o.decimalOdds)))
	}, 0.0, math.Inf(-1), math.Inf(1))
}

// LogarithmicOdds implements the "logarithmic" approach, which raises each
// implied probability to the power c.
func LogarithmicOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	// c runs over the positive numbers, solved for in its log as in OddsRatioOdds.
	return solveForC(odds, func(o Odds, logC float64) float64 {
		return math.Pow(1.0/o.decimalOdds, math.Exp(logC))
	}, 0.0, math.Inf(-1), math.Inf(1))
}

// ProbitOdds implements the "probit" approach, which moves every implied
// probability by the same c in probit space, the quantile of the standard
// normal: true probability = Phi(Phi^-1(p) - c). It is the odds ratio approach
// with the normal distribution in place of the logistic one.
func ProbitOdds(odds ...Odds) ([]Odds, error) {
	if len(odds) < 2 {
		return nil, fmt.Errorf("need at least two odds")
	}
	// Phi^-1(p) = sqrt(2) * erfinv(2p - 1) and Phi(x) = erfc(-x / sqrt(2)) / 2.
	return solveForC(odds, func(o Odds, c float64) float64 {
		z := math.Sqrt2 * math.Erfinv(2.0*o.ImpliedProb().decimal-1.0)
		return math.Erfc(-(z-c)/math.Sqrt2) / 2.0
	}, 0.0, math.Inf(-1), math.Inf(1))
}
