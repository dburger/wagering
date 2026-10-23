/*
Package wagering provides types for representing odds and performing computations
related to wagering.

Typical usage:

	likelihood := 0.6
	multiplier := 0.3
	bankroll := 10000.0
	odds, err := wagering.NewOddsFromAmerican(-110.0)
	if err != nil {
		return err
	}
	wager := odds.KellyStake(likelihood, multiplier, bankroll)

Note that when odds are constructed from american or decimal odds, that value is
held explicitly and the other format is computed but may suffer from minor rounding
skew.
*/
package wagering

import (
	"errors"
	"fmt"
	"math"
	"math/big"
)

// Odds is a price, in every format. Make one with a constructor: the zero Odds isn't a price, and
// it and any other Odds that isn't Valid make the calculations below return 0 (no bet, no edge),
// ArbTo false, Vig NaN and the devig functions an error, rather than a plausible wrong answer.
type Odds struct {
	decimalOdds   float64
	americanOdds  float64
	netFractional float64
	cents         float64
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
	Cents    = OddsFormat{"cents"}
)

func OddsFormatFromString(s string) (OddsFormat, error) {
	switch s {
	case American.slug:
		return American, nil
	case Decimal.slug:
		return Decimal, nil
	case Cents.slug:
		return Cents, nil
	default:
		return Unknown, fmt.Errorf("unknown odds format: %v", s)
	}
}

func (of OddsFormat) ToString() string {
	return of.slug
}

// Note that these constructors generally try to create the fields of
// the Odds struct directly from the provided value rather than computing
// one, say decimal, and then deriving the others from it. Presumably,
// this results in fewer rounding errors and more accurate odds.

// NewOdds constructs a new Odds from the given price and odds format.
func NewOdds(price float64, oddsFormat OddsFormat) (Odds, error) {
	if oddsFormat == American {
		return NewOddsFromAmerican(price)
	} else if oddsFormat == Decimal {
		return NewOddsFromDecimal(price)
	} else if oddsFormat == Cents {
		return NewOddsFromCents(price)
	} else {
		return Odds{}, fmt.Errorf("unknown odds format: %v", oddsFormat)
	}
}

// NewOddsFromAmerican constructs a new Odds from the given american odds, which must be finite
// and at most -100 or at least +100; anything else is an error. For whole-number american odds,
// the decimal odds are the float64 nearest their exact value.
func NewOddsFromAmerican(americanOdds float64) (Odds, error) {
	// Written as the valid range negated, so NaN, which fails every comparison, is rejected too.
	if !(americanOdds <= -100.0 || americanOdds >= 100.0) || math.IsInf(americanOdds, 0) {
		return Odds{}, fmt.Errorf("invalid american odds: %v", americanOdds)
	}
	var decimalOdds float64
	var netFractional float64
	if americanOdds >= 100.0 {
		decimalOdds = (americanOdds + 100.0) / 100.0
		netFractional = americanOdds / 100.0
	} else {
		decimalOdds = (americanOdds - 100.0) / americanOdds
		netFractional = -100.0 / americanOdds
	}
	var cents float64
	if americanOdds > 0.0 {
		cents = 10000.0 / (americanOdds + 100.0)
	} else {
		cents = (-americanOdds * 100.0) / (-americanOdds + 100.0)
	}
	return Odds{
		decimalOdds:   decimalOdds,
		americanOdds:  americanOdds,
		netFractional: netFractional,
		cents:         cents,
	}, nil
}

// NewOddsFromDecimal constructs a new Odds from the given decimal odds, which must be finite and
// greater than 1; anything else is an error. The american odds are the float64 nearest the exact
// value for the given decimal odds.
func NewOddsFromDecimal(decimalOdds float64) (Odds, error) {
	// Written as the valid range negated, so NaN, which fails every comparison, is rejected too.
	if !(decimalOdds > 1.0) || math.IsInf(decimalOdds, 0) {
		return Odds{}, fmt.Errorf("invalid decimal odds: %v", decimalOdds)
	}
	var americanOdds float64
	if decimalOdds >= 2.0 {
		americanOdds = (decimalOdds - 1.0) * 100.0
	} else {
		americanOdds = -100.0 / (decimalOdds - 1.0)
	}
	netFractional := decimalOdds - 1.0
	cents := 100.0 / decimalOdds
	return Odds{
		decimalOdds:   decimalOdds,
		americanOdds:  americanOdds,
		netFractional: netFractional,
		cents:         cents,
	}, nil
}

// NewOddsFromCents constructs a new Odds from the given price in cents of a contract paying 100
// cents, which must be greater than 0 and less than 100; anything else is an error. Every format is
// the float64 nearest its exact value for the given cents.
func NewOddsFromCents(cents float64) (Odds, error) {
	// Written as the valid range negated, so NaN, which fails every comparison, is rejected too.
	if !(cents > 0.0 && cents < 100.0) {
		return Odds{}, fmt.Errorf("invalid cents: %v", cents)
	}
	// A price in cents is the probability as a percent, exactly.
	return oddsFromProb(new(big.Rat).Quo(new(big.Rat).SetFloat64(cents), big.NewRat(100, 1))), nil
}

// NewOddsFromProbability constructs a new Odds from the probability of winning it implies, which
// must be above 0 and below 1; 0 and 1 are probabilities but not prices, and are an error. Every
// format is the float64 nearest its exact value for the given probability.
func NewOddsFromProbability(prob Probability) (Odds, error) {
	if !(prob.decimal > 0.0 && prob.decimal < 1.0) {
		return Odds{}, fmt.Errorf("invalid probability for odds: %v", prob.decimal)
	}
	return oddsFromProb(new(big.Rat).SetFloat64(prob.decimal)), nil
}

// oddsFromProbFloat is NewOddsFromProbability for a probability held as a float64, as the averages
// and devig methods compute them.
func oddsFromProbFloat(p float64) (Odds, error) {
	prob, err := NewProbabilityFromDecimal(p)
	if err != nil {
		return Odds{}, err
	}
	return NewOddsFromProbability(prob)
}

// oddsFromProb returns the Odds for the implied probability p, above 0 and below 1, with every
// format the float64 nearest its exact value. Each is computed exactly and rounded once: in
// float64, american and net fractional odds take two operations that each round, which missed
// the nearest float64 for more than half of all prices.
func oddsFromProb(p *big.Rat) Odds {
	hundred := big.NewRat(100, 1)
	nearest := func(r *big.Rat) float64 {
		f, _ := r.Float64()
		return f
	}
	lose := new(big.Rat).Sub(big.NewRat(1, 1), p) // the probability of losing
	net := new(big.Rat).Quo(lose, p)              // what 1 staked wins
	var american *big.Rat
	if p.Cmp(big.NewRat(1, 2)) <= 0 {
		// The underdog side, decimal odds of 2 or more: the win on 100 staked.
		american = new(big.Rat).Mul(net, hundred)
	} else {
		// The favorite: the stake to win 100, negative.
		american = new(big.Rat).Quo(new(big.Rat).Neg(hundred), net)
	}
	return Odds{
		decimalOdds:   nearest(new(big.Rat).Inv(p)),
		americanOdds:  nearest(american),
		netFractional: nearest(net),
		cents:         nearest(new(big.Rat).Mul(p, hundred)),
	}
}

// Valid returns whether odds is a price, as every Odds from a constructor is: decimal odds above
// 1. The zero Odds isn't.
func (odds Odds) Valid() bool {
	return odds.decimalOdds > 1.0 && !math.IsInf(odds.decimalOdds, 1)
}

// allValid returns whether every one of odds is Valid.
func allValid(odds ...Odds) bool {
	for _, o := range odds {
		if !o.Valid() {
			return false
		}
	}
	return true
}

// checkValid returns an error naming the first of odds that isn't Valid, for the devig functions.
func checkValid(odds ...Odds) error {
	for i, o := range odds {
		if !o.Valid() {
			return fmt.Errorf("invalid odds at %d: decimal %v", i, o.decimalOdds)
		}
	}
	return nil
}

// American returns the american odds.
func (odds Odds) American() float64 {
	return odds.americanOdds
}

// Decimal returns the decimal odds.
func (odds Odds) Decimal() float64 {
	return odds.decimalOdds
}

// Cents returns the price in cents of a contract paying 100 cents.
func (odds Odds) Cents() float64 {
	return odds.cents
}

// ToString returns odds in the given format: american as +200.00 or -200.00, decimal as 3.00,
// and cents as 33.33¢. Any other format, Unknown included, which OddsFormatFromString returns
// with its error, gives decimal: a price shown in the wrong format is still the right price.
func (odds Odds) ToString(of OddsFormat) string {
	if of == American {
		if odds.americanOdds >= 0 {
			return fmt.Sprintf("+%.2f", odds.americanOdds)
		} else {
			return fmt.Sprintf("%.2f", odds.americanOdds)
		}
	} else if of == Cents {
		return fmt.Sprintf("%.2f¢", odds.cents)
	} else {
		return fmt.Sprintf("%.2f", odds.decimalOdds)
	}
}

// AverageOdds provides a way to compute the average of a number of Odds. The average
// is computed as the average of the implied probabilities, weighted when Odds are given
// to AccumulateWeighted.
type AverageOdds struct {
	probSum   float64
	weightSum float64
	err       error // the first invalid Odds or weight accumulated, which Average returns.
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
// as Odds given to Accumulate. weight must be finite and not negative: 0 counts the odds for
// nothing, and anything else makes Average an error.
func (ao *AverageOdds) AccumulateWeighted(weight float64, odds ...Odds) {
	if ao.err == nil {
		// Written as the valid range negated, so NaN, which fails every comparison, is caught too.
		if !(weight >= 0.0) || math.IsInf(weight, 1) {
			ao.err = fmt.Errorf("invalid weight accumulated: %v", weight)
		} else if !allValid(odds...) {
			ao.err = errors.New("invalid odds accumulated")
		}
	}
	ao.probSum += weight * probSum(odds...)
	ao.weightSum += weight * float64(len(odds))
}

// Average returns the average Odds for the AverageOdds. It is an error if nothing has been
// accumulated, if any Odds or weight accumulated wasn't valid (the first one is reported), or if
// the average implied probability isn't below 1, which no decimal odds give.
func (ao *AverageOdds) Average() (Odds, error) {
	if ao.err != nil {
		return Odds{}, ao.err
	}
	if ao.weightSum == 0 {
		return Odds{}, errors.New("no odds to average")
	}
	return oddsFromProbFloat(ao.probSum / ao.weightSum)
}

func probs(odds ...Odds) []Probability {
	var ps []Probability
	for _, o := range odds {
		ps = append(ps, o.ImpliedProb())
	}
	return ps
}

// probSum returns the summation of the implied probabilities for the given odds.
func probSum(odds ...Odds) float64 {
	ps := probs(odds...)
	sum := 0.0
	for _, p := range ps {
		sum += p.decimal
	}
	return sum
}

func transOdds(prob func(Odds) float64, odds ...Odds) ([]Odds, error) {
	var trans []Odds
	for _, o := range odds {
		todds, err := oddsFromProbFloat(prob(o))
		if err != nil {
			return nil, fmt.Errorf("devigged odds: %w", err)
		}
		trans = append(trans, todds)
	}
	return trans, nil
}

// overround returns the sum of the implied probabilities of a market's outcomes, one Odds per
// outcome: 1 for a fair market, and above 1 by the book's margin. -110/-110 is 1.0476.
func overround(odds ...Odds) float64 {
	return probSum(odds...)
}

// margin returns how far a market's overround is above 1, the book's edge on the implied
// probabilities: -110/-110 has a margin of 0.0476, 4.76%.
func margin(odds ...Odds) float64 {
	return overround(odds...) - 1.0
}

// Vig returns the vigorish of a market priced at odds, one Odds per outcome: the book's expected
// share of all that is staked when the stakes pay the same on every outcome, as a fraction. It is
// the margin, the amount the implied probabilities sum to above 1, divided by that sum, the
// overround: -110/-110 has a margin of 4.76% and a vig of 4.55%. It is NaN if any of odds isn't
// Valid, so a comparison with it is false.
func Vig(odds ...Odds) float64 {
	if !allValid(odds...) {
		return math.NaN()
	}
	ov := overround(odds...)
	return (ov - 1.0) / ov
}

// KellyFraction returns the fraction of the bankroll to wager given the probability
// for success and kelly multiplier.
// https://en.wikipedia.org/wiki/Kelly_criterion
func (odds Odds) KellyFraction(prob Probability, mult float64) float64 {
	if !odds.Valid() {
		return 0.0
	}
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
	// Odds from a constructor are above 1, so the error can't happen. The zero Odds, which isn't
	// one, gets probability 0 (see AUDIT.md #1).
	prob, _ := NewProbabilityFromDecimal(1 / odds.decimalOdds)
	return prob
}

// ExpectedValueProb returns the long term expected value when wagering odds
// at the given probability. The result is given as the percent increase or
// decrease (negative) of the wager. It is 0 for odds that aren't Valid.
func (odds Odds) ExpectedValueProb(prob Probability) float64 {
	if !odds.Valid() {
		return 0.0
	}
	return prob.decimal*(odds.decimalOdds-1.0) - (1.0 - prob.decimal)
}

// ExpectedValueOdds returns the long term expected value when wagering odds
// at the given true odds.  The result is given as the percent increase or
// decrease (negative) of the wager. It is 0 if either isn't Valid.
func (odds Odds) ExpectedValueOdds(trueOdds Odds) float64 {
	if !trueOdds.Valid() {
		return 0.0
	}
	return odds.ExpectedValueProb(trueOdds.ImpliedProb())
}

// Meg returns the maximum expected growth, in basis points: the compounded growth
// of the bankroll per bet from betting the full Kelly fraction at odds when
// trueOdds are the real odds, (1 + f*b)^p * (1 - f)^q - 1. It is 0 without an
// edge, since Kelly then bets nothing, and if either isn't Valid.
// https://en.wikipedia.org/wiki/Kelly_criterion
func (odds Odds) Meg(trueOdds Odds) float64 {
	if !odds.Valid() || !trueOdds.Valid() {
		return 0.0
	}
	prob := trueOdds.ImpliedProb()
	fraction := odds.KellyFraction(prob, 1.0)
	p := prob.decimal
	q := 1.0 - p
	// Computed through the log growth, which Log1p and Expm1 keep accurate for
	// small fractions.
	logGrowth := p*math.Log1p(fraction*odds.netFractional) + q*math.Log1p(-fraction)
	return 10000.0 * math.Expm1(logGrowth)
}

// ArbTo returns whether the given odds is an arbitrage to other odds, never if either isn't Valid.
func (odds Odds) ArbTo(other Odds) bool {
	if !odds.Valid() || !other.Valid() {
		return false
	}
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

// ArbRoi returns the return on the total staked from betting odds and other with stakes that pay
// the same either way, as a fraction: negative if they aren't an arbitrage, and 0 if either isn't
// Valid.
func (odds Odds) ArbRoi(other Odds) float64 {
	if !odds.Valid() || !other.Valid() {
		return 0.0
	}
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
// between passing the decimal or percent is removed. The constructors accept only
// a probability from 0 to 1 (0 to 100 percent), so a Probability always holds one.
type Probability struct {
	decimal float64
	percent float64
}

// NewProbabilityFromPercent constructs a Probability from the given percent, which must be from 0
// to 100; anything else is an error.
func NewProbabilityFromPercent(percent float64) (Probability, error) {
	// Written as the valid range negated, so NaN, which fails every comparison, is rejected too.
	if !(percent >= 0.0 && percent <= 100.0) {
		return Probability{}, fmt.Errorf("invalid percent probability: %v", percent)
	}
	return Probability{percent / 100.0, percent}, nil
}

// NewProbabilityFromDecimal constructs a Probability from the given decimal, which must be from 0
// to 1; anything else is an error.
func NewProbabilityFromDecimal(decimal float64) (Probability, error) {
	// Written as the valid range negated, so NaN, which fails every comparison, is rejected too.
	if !(decimal >= 0.0 && decimal <= 1.0) {
		return Probability{}, fmt.Errorf("invalid decimal probability: %v", decimal)
	}
	return Probability{decimal, decimal * 100.0}, nil
}

// Decimal returns the probability as a decimal, from 0 to 1.
func (p Probability) Decimal() float64 {
	return p.decimal
}

// Percent returns the probability as a percent, from 0 to 100.
func (p Probability) Percent() float64 {
	return p.percent
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
	if err := checkValid(odds...); err != nil {
		return nil, err
	}
	ov := overround(odds...)
	var norms []Odds
	for _, o := range odds {
		nodds, err := NewOddsFromDecimal(o.decimalOdds * ov)
		if err != nil {
			return nil, fmt.Errorf("equal margin odds: %w", err)
		}
		norms = append(norms, nodds)
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
	if err := checkValid(odds...); err != nil {
		return nil, err
	}
	n := float64(len(odds))
	m := margin(odds...)
	var norms []Odds
	for _, o := range odds {
		prob := 1/o.decimalOdds - m/n
		if prob <= 0.0 {
			return nil, fmt.Errorf("margin %v exceeds the implied probability of decimal odds %v", m, o.decimalOdds)
		}
		nodds, err := oddsFromProbFloat(prob)
		if err != nil {
			return nil, fmt.Errorf("MPT odds: %w", err)
		}
		norms = append(norms, nodds)
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
	return transOdds(func(o Odds) float64 { return prob(o, c) }, odds...)
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
	if err := checkValid(odds...); err != nil {
		return nil, err
	}
	ov := overround(odds...)

	return solveForC(odds, func(o Odds, c float64) float64 {
		// Only the pi^2 term is divided by the overround, as in Jullien and Salanie
		// (1994) and Strumbelj (2014).
		sqrt := math.Sqrt(math.Pow(c, 2.0) + 4.0*(1.0-c)*math.Pow(o.ImpliedProb().decimal, 2.0)/ov)
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
	if err := checkValid(odds...); err != nil {
		return nil, err
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
	if err := checkValid(odds...); err != nil {
		return nil, err
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
	if err := checkValid(odds...); err != nil {
		return nil, err
	}
	// Phi^-1(p) = sqrt(2) * erfinv(2p - 1) and Phi(x) = erfc(-x / sqrt(2)) / 2.
	return solveForC(odds, func(o Odds, c float64) float64 {
		z := math.Sqrt2 * math.Erfinv(2.0*o.ImpliedProb().decimal-1.0)
		return math.Erfc(-(z-c)/math.Sqrt2) / 2.0
	}, 0.0, math.Inf(-1), math.Inf(1))
}
