package wagering

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertAmerican(t *testing.T) {
	var expectedOdds = []struct {
		americanOdds        float64
		expectedDecimalOdds float64
	}{
		{+9900.0, 100.0},
		{+300.0, 4.0},
		{+150.0, 2.5},
		{-110.0, 1.91},
		{-150.0, 1.67},
		{-300.0, 1.33},
		{-1000.0, 1.1},
	}
	for _, odds := range expectedOdds {
		converted := NewOddsFromAmerican(odds.americanOdds)
		assert.Equal(t, odds.americanOdds, converted.americanOdds, "converting american %v", odds.americanOdds)
		assert.InDeltaf(t, odds.expectedDecimalOdds, converted.decimalOdds, 0.01, "converting american %v", odds.americanOdds)
	}
}

func TestConvertDecimal(t *testing.T) {
	var expectedOdds = []struct {
		decimalOdds          float64
		expectedAmericanOdds float64
	}{
		{100.0, +9900.0},
		{4.0, +300.0},
		{2.5, +150.0},
		{1.91, -109.89},
		{1.67, -149.25},
		{1.33, -303.03},
		{1.1, -1000.0},
	}
	for _, odds := range expectedOdds {
		converted := NewOddsFromDecimal(odds.decimalOdds)
		assert.InDeltaf(t, odds.expectedAmericanOdds, converted.americanOdds, 0.01, "converting decimal %v", odds.decimalOdds)
		assert.Equal(t, odds.decimalOdds, converted.decimalOdds, "converting decimal %v", odds.decimalOdds)
	}
}

func TestImpliedProbability(t *testing.T) {
	var expectedProbabilities = []struct {
		odds Odds
		prob float64
	}{
		{NewOddsFromDecimal(100.0), 1.0},
		{NewOddsFromDecimal(4.0), 25.0},
		{NewOddsFromDecimal(2.5), 40.0},
		{NewOddsFromDecimal(1.91), 52.35},
		{NewOddsFromDecimal(1.67), 59.88},
		{NewOddsFromDecimal(1.33), 75.18},
		{NewOddsFromDecimal(1.1), 90.90},
	}
	for _, ep := range expectedProbabilities {
		assert.InDeltaf(t, ep.prob, ep.odds.ImpliedProb().percent, 0.01, "converting decimal %v", ep.odds.decimalOdds)
	}
}

func TestOdds_KellyFraction(t *testing.T) {
	odds := NewOddsFromDecimal(2.0)
	prob := NewProbabilityFromDecimal(0.6)
	mult := 1.0
	fraction := odds.KellyFraction(prob, mult)
	assert.InDeltaf(t, 0.2, fraction, 0.01, "calculating kelly value for %v decimal odds with prob %v and multiplier %v", odds.decimalOdds, prob.percent, mult)
}

func TestOdds_KellyStake(t *testing.T) {
	odds := NewOddsFromAmerican(200.0)
	prob := NewProbabilityFromPercent(60.0)
	mult := 0.25
	wager := odds.KellyStake(prob, mult, 1000.00)
	assert.InDeltaf(t, 100, wager, 0.1, "calculating wager for %v decimal odds with prob %v and multiplier %v", odds.decimalOdds, prob.percent, mult)
}

func TestOdds_Equals(t *testing.T) {
	odds1 := NewOddsFromDecimal(1.5)
	odds2 := NewOddsFromDecimal(1.5)
	odds3 := NewOddsFromDecimal(2.0)

	assert.True(t, odds1.Equals(odds2))
	assert.False(t, odds2.Equals(odds3))
}

func TestOdds_Longer(t *testing.T) {
	odds1 := NewOddsFromDecimal(1.5)
	odds2 := NewOddsFromDecimal(1.5)
	odds3 := NewOddsFromDecimal(2.0)
	assert.True(t, odds3.Longer(odds1))
	assert.False(t, odds2.Longer(odds1))
}

func TestOdds_Shorter(t *testing.T) {
	odds1 := NewOddsFromDecimal(1.5)
	odds2 := NewOddsFromDecimal(1.5)
	odds3 := NewOddsFromDecimal(2.0)
	assert.True(t, odds1.Shorter(odds3))
	assert.False(t, odds1.Shorter(odds2))
}

func TestOdds_ExpectedValueProb(t *testing.T) {
	odds := NewOddsFromAmerican(-110.0)
	prob := NewProbabilityFromPercent(50.0)
	ev := odds.ExpectedValueProb(prob)
	assert.InDeltaf(t, -0.0455, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)

	odds = NewOddsFromAmerican(+180.0)
	prob = NewProbabilityFromPercent(30.0)
	ev = odds.ExpectedValueProb(prob)
	assert.InDeltaf(t, -0.16, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)
}

func TestOdds_ExpectedValueOdds(t *testing.T) {
	odds := NewOddsFromAmerican(-110.0)
	trueOdds := NewOddsFromAmerican(+100.0)
	ev := odds.ExpectedValueOdds(trueOdds)
	assert.InDeltaf(t, -0.0455, ev, 0.001, "expected value of %v at %v% odds", odds.americanOdds, trueOdds.Decimal())

	odds = NewOddsFromAmerican(+180.0)
	trueOdds = NewOddsFromAmerican(+233.0)
	ev = odds.ExpectedValueOdds(trueOdds)
	assert.InDeltaf(t, -0.16, ev, 0.001, "expected value of %v at %v% odds", odds.americanOdds, trueOdds.Decimal())
}

func TestOdds_ToString(t *testing.T) {
	odds := NewOddsFromAmerican(+200.0)
	assert.Equal(t, "+200.00", odds.ToString(American))
	assert.Equal(t, "3.00", odds.ToString(Decimal))
}

func TestOddsFormat_ToString(t *testing.T) {
	assert.Equal(t, "american", American.ToString())
	assert.Equal(t, "decimal", Decimal.ToString())
}

func TestMarketWidth(t *testing.T) {
	odds1 := NewOddsFromAmerican(-141.0)
	odds2 := NewOddsFromAmerican(+123.0)
	assert.Equal(t, 18.0, MarketWidth(odds1, odds2))

	odds1 = NewOddsFromAmerican(-110.0)
	odds2 = NewOddsFromAmerican(-114.0)
	assert.Equal(t, 24.0, MarketWidth(odds1, odds2))

	odds1 = NewOddsFromAmerican(+150.0)
	odds2 = NewOddsFromAmerican(+137.0)
	assert.Equal(t, -87.0, MarketWidth(odds1, odds2))
}

func TestProbabilityConstruction(t *testing.T) {
	prob := NewProbabilityFromDecimal(0.5)
	assert.Equal(t, 0.5, prob.decimal)
	assert.Equal(t, 50.0, prob.percent)

	prob = NewProbabilityFromPercent(50.0)
	assert.Equal(t, 0.5, prob.decimal)
	assert.Equal(t, 50.0, prob.percent)
}

func dummyAverageOdds() AverageOdds {
	ao := NewAverageOdds()
	ao.Accumulate(NewOddsFromDecimal(3.0))
	ao.Accumulate(NewOddsFromDecimal(5.0))
	ao.Accumulate(NewOddsFromDecimal(7.0))
	return ao
}

func TestAverageOdds(t *testing.T) {
	ao := dummyAverageOdds()
	assert.Equal(t, 5.0, ao.Average().decimalOdds)
}

func TestAverageOdds_AverageWithout(t *testing.T) {
	ao := dummyAverageOdds()
	assert.Equal(t, 4.0, ao.AverageWithout(NewOddsFromDecimal(7.0), 1).decimalOdds)
	assert.Equal(t, 10.0, ao.AverageWithout(NewOddsFromDecimal(2.5), 2).decimalOdds)
}

func round(value float64, places uint) float64 {
	mult := math.Pow(10, float64(places))
	return math.Round(value*mult) / mult
}

// sampleOdds1 returns the Odds from the example at
// https://winnerodds.com/valuebettingblog/true-odds-calculator/
// for win, draw, win for Real Madrid versus Aletico de Madrid.
func sampleOdds1() []Odds {
	return []Odds{NewOddsFromDecimal(2.09), NewOddsFromDecimal(3.59), NewOddsFromDecimal(3.77)}
}

// sampleOdds2 returns the Odds from the tests at
// https://github.com/mberk/shin/blob/master/tests/test_shin.py.
func sampleOdds2() []Odds {
	return []Odds{NewOddsFromDecimal(2.6), NewOddsFromDecimal(2.4), NewOddsFromDecimal(4.3)}
}

func TestEqualMarginOdds(t *testing.T) {
	trueOdds := EqualMarginOdds(sampleOdds1()...)
	assert.Equal(t, 2.1365, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6700, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8540, round(trueOdds[2].decimalOdds, 4))
}

func TestAdditiveOdds(t *testing.T) {
	trueOdds := AdditiveOdds(sampleOdds1()...)
	assert.Equal(t, 2.1229, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6883, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8786, round(trueOdds[2].decimalOdds, 4))
}

func TestMPTOOdds(t *testing.T) {
	trueOdds := MPTOdds(sampleOdds1()...)
	assert.Equal(t, 2.1229, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6883, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8786, round(trueOdds[2].decimalOdds, 4))
}

func TestShinOdds(t *testing.T) {
	trueOdds := ShinOdds(sampleOdds1()...)
	assert.Equal(t, 2.1264, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6836, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8723, round(trueOdds[2].decimalOdds, 4))

	trueOdds = ShinOdds(sampleOdds2()...)
	assert.Equal(t, 0.372994, round(trueOdds[0].ImpliedProb().decimal, 6))
	assert.Equal(t, 0.40478, round(trueOdds[1].ImpliedProb().decimal, 6))
	assert.Equal(t, 0.222226, round(trueOdds[2].ImpliedProb().decimal, 6))
}

func TestOddsRatioOdds(t *testing.T) {
	trueOdds := OddsRatioOdds(sampleOdds1()...)
	assert.Equal(t, 2.1285, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6814, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8678, round(trueOdds[2].decimalOdds, 4))
}

func TestLogarithmicOdds(t *testing.T) {
	trueOdds := LogarithmicOdds(sampleOdds1()...)
	assert.Equal(t, 2.1230, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6888, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8778, round(trueOdds[2].decimalOdds, 4))
}
