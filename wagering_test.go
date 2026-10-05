package wagering

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOdds(t *testing.T) {
	odds, err := NewOdds(2.0, Decimal)
	assert.Nil(t, err)
	assert.Equal(t, 2.0, odds.decimalOdds)
	assert.Equal(t, 100.0, odds.americanOdds)

	odds, err = NewOdds(150, American)
	assert.Nil(t, err)
	assert.Equal(t, 2.5, odds.decimalOdds)
	assert.Equal(t, 150.0, odds.americanOdds)

	odds, err = NewOdds(100, Unknown)
	assert.NotNil(t, err)
}

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

func TestOdds_ArbTo(t *testing.T) {
	odds1 := NewOddsFromAmerican(-110.0)
	odds2 := NewOddsFromAmerican(-110.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = NewOddsFromAmerican(100.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = NewOddsFromAmerican(109.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = NewOddsFromAmerican(110.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = NewOddsFromAmerican(111.0)
	assert.True(t, odds1.ArbTo(odds2))
	assert.True(t, odds2.ArbTo(odds1))
}

func TestOdds_ArbRoi(t *testing.T) {
	odds1 := NewOddsFromDecimal(3.0)
	odds2 := NewOddsFromDecimal(3.0)
	assert.Equal(t, 0.5, odds1.ArbRoi(odds2))

	odds1 = NewOddsFromAmerican(+150)
	odds2 = NewOddsFromAmerican(+150)
	assert.Equal(t, 0.25, odds1.ArbRoi(odds2))

	odds1 = NewOddsFromAmerican(-200)
	odds2 = NewOddsFromAmerican(-200)
	assert.Equal(t, -0.25, odds1.ArbRoi(odds2))
}

func TestOdds_ToString(t *testing.T) {
	odds := NewOddsFromAmerican(+200.0)
	assert.Equal(t, "+200.00", odds.ToString(American))
	assert.Equal(t, "3.00", odds.ToString(Decimal))

	odds = NewOddsFromAmerican(-200.0)
	assert.Equal(t, "-200.00", odds.ToString(American))
}

func TestOddsFormat_ToString(t *testing.T) {
	assert.Equal(t, "american", American.ToString())
	assert.Equal(t, "decimal", Decimal.ToString())
}

func TestOddsFormatFromString(t *testing.T) {
	format, err := OddsFormatFromString("american")
	assert.Equal(t, format, American)
	assert.Nil(t, err)

	format, err = OddsFormatFromString("decimal")
	assert.Equal(t, format, Decimal)
	assert.Nil(t, err)

	format, err = OddsFormatFromString("foo")
	assert.Equal(t, format, Unknown)
	assert.NotNil(t, err)
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
	// 4.43661971844
	assert.InDeltaf(t, 4.4366, ao.Average().decimalOdds, 0.00005, "averaging odds %v", ao)
}

func TestAverageOddsWeighted(t *testing.T) {
	tests := []struct {
		name       string
		accumulate func(ao *AverageOdds)
		want       float64
	}{
		{
			// (2/3 + 1/5) / 3 = 0.28889
			name: "weight of two",
			accumulate: func(ao *AverageOdds) {
				ao.AccumulateWeighted(2, NewOddsFromDecimal(3.0))
				ao.AccumulateWeighted(1, NewOddsFromDecimal(5.0))
			},
			want: 3.4615,
		},
		{
			// (1/7 + 3/3) / 4 = 0.28571, with Accumulate weighing one.
			name: "mixed with Accumulate",
			accumulate: func(ao *AverageOdds) {
				ao.Accumulate(NewOddsFromDecimal(7.0))
				ao.AccumulateWeighted(3, NewOddsFromDecimal(3.0))
			},
			want: 3.5,
		},
		{
			// (0.5/2 + 0.5/4) / 1 = 0.375, weights need not be whole.
			name: "fractional weights",
			accumulate: func(ao *AverageOdds) {
				ao.AccumulateWeighted(0.5, NewOddsFromDecimal(2.0), NewOddsFromDecimal(4.0))
			},
			want: 2.6667,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ao := NewAverageOdds()
			tt.accumulate(&ao)
			assert.InDeltaf(t, tt.want, ao.Average().decimalOdds, 0.00005, "averaging odds %v", ao)
		})
	}
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

// tenRunners returns a field of ten runners at a 20% margin, as a race or a
// futures market has. The old fixed point solver never converged on it for
// Shin or the logarithmic method.
func tenRunners() []Odds {
	return decimalOdds(11.14, 5.97, 52.76, 10.9, 5.76, 5.11, 10.44, 10.55, 11.02, 5.51)
}

// underround returns a two way market whose implied probabilities sum to
// 95.6%, which the solvers must push the other way.
func underround() []Odds {
	return decimalOdds(1.8, 2.5)
}

func decimalOdds(prices ...float64) []Odds {
	var odds []Odds
	for _, p := range prices {
		odds = append(odds, NewOddsFromDecimal(p))
	}
	return odds
}

// assertProbs asserts the implied probabilities of odds to seven places.
func assertProbs(t *testing.T, want []float64, odds []Odds) {
	t.Helper()
	var got []float64
	for _, o := range odds {
		got = append(got, round(o.ImpliedProb().decimal, 7))
	}
	assert.Equal(t, want, got)
}

func TestEqualMarginOdds(t *testing.T) {
	trueOdds, err := EqualMarginOdds()
	assert.NotNil(t, err)

	trueOdds, err = EqualMarginOdds(NewOddsFromDecimal(1.0))
	assert.NotNil(t, err)

	trueOdds, err = EqualMarginOdds(sampleOdds1()...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1365, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6700, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8540, round(trueOdds[2].decimalOdds, 4))
}

func TestMPTOdds(t *testing.T) {
	trueOdds, err := MPTOdds()
	assert.NotNil(t, err)

	trueOdds, err = MPTOdds(NewOddsFromDecimal(1.0))
	assert.NotNil(t, err)

	trueOdds, err = MPTOdds(sampleOdds1()...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1229, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6883, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8786, round(trueOdds[2].decimalOdds, 4))

	// It takes margin / n from each implied probability, so a long enough price
	// in a wide enough book has more taken than it has. An 18.3% margin over
	// three prices takes 6.1% from the +1900's 5%. The others then sum to over
	// one, so none of them is fair either.
	trueOdds, err = MPTOdds(NewOddsFromAmerican(-400), NewOddsFromAmerican(200), NewOddsFromAmerican(1900))
	assert.Nil(t, trueOdds)
	assert.NotNil(t, err)

	// A 5% margin takes 1.7% from the same longshot.
	trueOdds, err = MPTOdds(NewOddsFromAmerican(-300), NewOddsFromAmerican(300), NewOddsFromAmerican(1900))
	assert.Nil(t, err)
	assertProbs(t, []float64{0.7333333, 0.2333333, 0.0333333}, trueOdds)
}

func TestShinOdds(t *testing.T) {
	trueOdds, err := ShinOdds()
	assert.NotNil(t, err)

	trueOdds, err = ShinOdds(NewOddsFromDecimal(1.0))
	assert.NotNil(t, err)

	trueOdds, err = ShinOdds(sampleOdds1()...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1264, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6836, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8723, round(trueOdds[2].decimalOdds, 4))

	// mberk/shin's test gives these to seven places.
	trueOdds, err = ShinOdds(sampleOdds2()...)
	assert.Nil(t, err)
	assert.Equal(t, 0.3729941, round(trueOdds[0].ImpliedProb().decimal, 7))
	assert.Equal(t, 0.4047794, round(trueOdds[1].ImpliedProb().decimal, 7))
	assert.Equal(t, 0.2222265, round(trueOdds[2].ImpliedProb().decimal, 7))

	trueOdds, err = ShinOdds(tenRunners()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0719737, 0.1433818, 0.0093293, 0.0737813, 0.1490059, 0.1693519, 0.0774805, 0.0765663, 0.0728675, 0.1562619}, trueOdds)

	trueOdds, err = ShinOdds(underround()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5777778, 0.4222222}, trueOdds)
}

func TestOddsRatioOdds(t *testing.T) {
	trueOdds, err := OddsRatioOdds()
	assert.NotNil(t, err)

	trueOdds, err = OddsRatioOdds(NewOddsFromDecimal(1.0))
	assert.NotNil(t, err)

	trueOdds, err = OddsRatioOdds(sampleOdds1()...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1285, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6814, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8678, round(trueOdds[2].decimalOdds, 4))

	trueOdds, err = OddsRatioOdds(tenRunners()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0740145, 0.1402120, 0.0154173, 0.0756730, 0.1454978, 0.1647182, 0.0790689, 0.0782294, 0.0748345, 0.1523345}, trueOdds)

	trueOdds, err = OddsRatioOdds(underround()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5779355, 0.4220645}, trueOdds)

	// A price of 1.0 has an implied probability of one at every c, so no c
	// brings the sum down to one.
	_, err = OddsRatioOdds(decimalOdds(1.0, 2.0)...)
	assert.EqualError(t, err, "no true odds for decimal odds [1 2]")
}

func TestLogarithmicOdds(t *testing.T) {
	trueOdds, err := LogarithmicOdds()
	assert.NotNil(t, err)

	trueOdds, err = LogarithmicOdds(NewOddsFromDecimal(1.0))
	assert.NotNil(t, err)

	trueOdds, err = LogarithmicOdds(sampleOdds1()...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1230, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6888, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8778, round(trueOdds[2].decimalOdds, 4))

	trueOdds, err = LogarithmicOdds(tenRunners()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0720491, 0.1423145, 0.0132009, 0.0737819, 0.1479856, 0.1686414, 0.0773364, 0.0764569, 0.0729057, 0.1553274}, trueOdds)

	trueOdds, err = LogarithmicOdds(underround()...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5763815, 0.4236185}, trueOdds)

	// A price of 1.0 has an implied probability of one at every c, so no c
	// brings the sum down to one.
	_, err = LogarithmicOdds(decimalOdds(1.0, 2.0)...)
	assert.EqualError(t, err, "no true odds for decimal odds [1 2]")
}

func TestOdds_Meg(t *testing.T) {
	// Expected values are the compounded growth per bet of a full Kelly bet, in basis
	// points: 10000 * ((1 + f*b)^p * (1 - f)^(1 - p) - 1), with f the Kelly fraction.
	tests := []struct {
		name        string
		decimalOdds float64
		prob        float64
		want        float64
	}{
		{name: "even odds, 10% edge", decimalOdds: 2.0, prob: 0.55, want: 50.209297},
		{name: "long odds, 20% edge", decimalOdds: 4.0, prob: 0.30, want: 64.219901},
		{name: "short odds, 5% edge", decimalOdds: 1.5, prob: 0.70, want: 25.482014},
		{name: "small edge", decimalOdds: 2.0, prob: 0.501, want: 0.020000},
		{name: "no edge", decimalOdds: 2.0, prob: 0.5, want: 0},
		{name: "negative edge", decimalOdds: 2.0, prob: 0.45, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			odds := NewOddsFromDecimal(tt.decimalOdds)
			trueOdds := NewOddsFromDecimal(1 / tt.prob)
			assert.InDelta(t, tt.want, odds.Meg(trueOdds), 1e-5)
		})
	}
}
