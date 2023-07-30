package wagering

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestOddsEquals(t *testing.T) {
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

func TestExpectedValuePercent(t *testing.T) {
	odds := NewOddsFromAmerican(-110.0)
	prob := NewProbabilityFromPercent(50.0)
	ev := odds.ExpectedValuePercent(prob)
	assert.InDeltaf(t, -0.0455, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)

	odds = NewOddsFromAmerican(+180.0)
	prob = NewProbabilityFromPercent(30.0)
	ev = odds.ExpectedValuePercent(prob)
	assert.InDeltaf(t, -0.16, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)
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
	ao := AverageOdds{}
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

func TestTrueOddsNormalized(t *testing.T) {
	odds1 := NewOddsFromAmerican(-110)
	trueOdds := TrueOddsNormalized(odds1, odds1)
	assert.Equal(t, 100.0, trueOdds[0].americanOdds)
	assert.Equal(t, 100.0, trueOdds[1].americanOdds)

	odds2 := NewOddsFromAmerican(-121)
	odds3 := NewOddsFromAmerican(160)
	trueOdds = TrueOddsNormalized(odds2, odds3)
	assert.InDeltaf(t, -142.35, trueOdds[0].americanOdds, 0.005, "expected value of %v from %v/%v", odds2.americanOdds, odds2.americanOdds, odds3.americanOdds)
	assert.InDeltaf(t, +142.35, trueOdds[1].americanOdds, 0.005, "expected value of %v from %v/%v", odds3.americanOdds, odds2.americanOdds, odds3.americanOdds)

	odds1 = NewOddsFromDecimal(2)
	odds2 = NewOddsFromDecimal(3)
	odds3 = NewOddsFromDecimal(4)
	trueOdds = TrueOddsNormalized(odds1, odds2, odds3)
	assert.InDeltaf(t, 2.17, trueOdds[0].decimalOdds, 0.005, "expoected value of %v from %v/%v/%v", odds1.decimalOdds, odds1.decimalOdds, odds2.decimalOdds, odds3.decimalOdds)
	assert.InDeltaf(t, 3.25, trueOdds[1].decimalOdds, 0.005, "expoected value of %v from %v/%v/%v", odds2.decimalOdds, odds1.decimalOdds, odds2.decimalOdds, odds3.decimalOdds)
	assert.InDeltaf(t, 4.33, trueOdds[2].decimalOdds, 0.005, "expoected value of %v from %v/%v/%v", odds3.decimalOdds, odds1.decimalOdds, odds2.decimalOdds, odds3.decimalOdds)
}
