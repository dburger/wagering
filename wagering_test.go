package wagering

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// american returns the Odds for american odds, failing the test if they're invalid.
func american(t *testing.T, americanOdds float64) Odds {
	t.Helper()
	odds, err := NewOddsFromAmerican(americanOdds)
	if err != nil {
		t.Fatal(err)
	}
	return odds
}

// decimal returns the Odds for decimal odds, failing the test if they're invalid.
func decimal(t *testing.T, decimalOdds float64) Odds {
	t.Helper()
	odds, err := NewOddsFromDecimal(decimalOdds)
	if err != nil {
		t.Fatal(err)
	}
	return odds
}

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

	_, err = NewOdds(50, American)
	assert.EqualError(t, err, "invalid american odds: 50")

	_, err = NewOdds(1.0, Decimal)
	assert.EqualError(t, err, "invalid decimal odds: 1")

	odds, err = NewOdds(40, Cents)
	assert.Nil(t, err)
	assert.Equal(t, 2.5, odds.decimalOdds)
	assert.Equal(t, 150.0, odds.americanOdds)
	assert.Equal(t, 40.0, odds.cents)

	odds, err = NewOdds(150, American)
	assert.Nil(t, err)
	assert.Equal(t, 40.0, odds.cents)

	odds, err = NewOdds(4.0, Decimal)
	assert.Nil(t, err)
	assert.Equal(t, 25.0, odds.cents)

	_, err = NewOdds(100, Cents)
	assert.EqualError(t, err, "invalid cents: 100")
}

func TestNewOddsFromAmericanInvalid(t *testing.T) {
	for _, a := range []float64{0, 50, 99.99, -99.99, -50, math.Inf(1), math.Inf(-1), math.NaN()} {
		_, err := NewOddsFromAmerican(a)
		assert.Errorf(t, err, "american %v", a)
	}
	for _, a := range []float64{100, -100} {
		odds, err := NewOddsFromAmerican(a)
		assert.Nil(t, err)
		assert.Equal(t, 2.0, odds.decimalOdds, "american %v is even money", a)
	}
}

func TestNewOddsFromDecimalInvalid(t *testing.T) {
	for _, d := range []float64{1.0, 0.5, 0, -2, math.Inf(1), math.Inf(-1), math.NaN()} {
		_, err := NewOddsFromDecimal(d)
		assert.Errorf(t, err, "decimal %v", d)
	}
}

// TestNewOddsFromAmericanExact checks that whole-number american odds give the float64 nearest
// their exact decimal odds and net fractional odds: each is one rounding, of exact integers.
// a/100 + 1 and 1 - 100/a round twice, and miss for 154 of the 1,802 prices from -1000 to +1000,
// such as -150.
func TestNewOddsFromAmericanExact(t *testing.T) {
	nearest := func(r *big.Rat) float64 {
		f, _ := r.Float64()
		return f
	}
	for a := int64(-20000); a <= 20000; a++ {
		if a > -100 && a < 100 {
			continue
		}
		var dec, net *big.Rat
		if a > 0 {
			dec, net = big.NewRat(a+100, 100), big.NewRat(a, 100)
		} else {
			dec, net = big.NewRat(a-100, a), big.NewRat(-100, a)
		}
		odds := american(t, float64(a))
		if odds.decimalOdds != nearest(dec) || odds.netFractional != nearest(net) {
			t.Errorf("american %d: decimal %v, net %v, want %v, %v", a, odds.decimalOdds, odds.netFractional, nearest(dec), nearest(net))
		}
	}
}

// TestNewOddsFromDecimalExact checks that decimal odds give the float64 nearest their exact
// american odds: decimalOdds - 1 is exact, so only the multiply or divide rounds.
func TestNewOddsFromDecimalExact(t *testing.T) {
	for i := 0; i < 20000; i++ {
		d := 1.0001 + float64(i)*0.000731
		exact := new(big.Rat).Sub(new(big.Rat).SetFloat64(d), big.NewRat(1, 1))
		if d >= 2 {
			exact.Mul(exact, big.NewRat(100, 1))
		} else {
			exact.Quo(big.NewRat(-100, 1), exact)
		}
		want, _ := exact.Float64()
		if got := decimal(t, d).americanOdds; got != want {
			t.Errorf("decimal %v: american %v, want %v", d, got, want)
		}
	}
}

func TestNewOddsFromCentsInvalid(t *testing.T) {
	for _, c := range []float64{100, 150, 0, -20, math.Inf(1), math.Inf(-1), math.NaN()} {
		_, err := NewOddsFromCents(c)
		assert.Errorf(t, err, "cents %v", c)
	}
}

func TestConvertCents(t *testing.T) {
	var expectedOdds = []struct {
		cents                float64
		expectedDecimalOdds  float64
		expectedAmericanOdds float64
	}{
		{1, 100.0, +9900.0},
		{21.75, 4.5977, +359.77},
		{25, 4.0, +300.0},
		{50, 2.0, +100.0},
		{60, 1.6667, -150.0},
		{72, 1.3889, -257.14},
		{99.5, 1.005, -19900.0},
	}
	for _, odds := range expectedOdds {
		converted, err := NewOddsFromCents(odds.cents)
		assert.Nil(t, err)
		assert.InDeltaf(t, odds.expectedDecimalOdds, converted.decimalOdds, 0.0001, "converting cents %v", odds.cents)
		assert.InDeltaf(t, odds.expectedAmericanOdds, converted.americanOdds, 0.01, "converting cents %v", odds.cents)
		assert.Equal(t, odds.cents, converted.Cents(), "cents %v kept as given", odds.cents)
	}
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
		converted := american(t, odds.americanOdds)
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
		converted := decimal(t, odds.decimalOdds)
		assert.InDeltaf(t, odds.expectedAmericanOdds, converted.americanOdds, 0.01, "converting decimal %v", odds.decimalOdds)
		assert.Equal(t, odds.decimalOdds, converted.decimalOdds, "converting decimal %v", odds.decimalOdds)
	}
}

func TestImpliedProbability(t *testing.T) {
	var expectedProbabilities = []struct {
		decimalOdds float64
		prob        float64
	}{
		{100.0, 1.0},
		{4.0, 25.0},
		{2.5, 40.0},
		{1.91, 52.35},
		{1.67, 59.88},
		{1.33, 75.18},
		{1.1, 90.90},
	}
	for _, ep := range expectedProbabilities {
		odds := decimal(t, ep.decimalOdds)
		assert.InDeltaf(t, ep.prob, odds.ImpliedProb().percent, 0.01, "converting decimal %v", ep.decimalOdds)
	}
}

func TestOdds_KellyFraction(t *testing.T) {
	odds := decimal(t, 2.0)
	prob := NewProbabilityFromDecimal(0.6)
	mult := 1.0
	fraction := odds.KellyFraction(prob, mult)
	assert.InDeltaf(t, 0.2, fraction, 0.01, "calculating kelly value for %v decimal odds with prob %v and multiplier %v", odds.decimalOdds, prob.percent, mult)
}

func TestOdds_KellyStake(t *testing.T) {
	odds := american(t, 200.0)
	prob := NewProbabilityFromPercent(60.0)
	mult := 0.25
	wager := odds.KellyStake(prob, mult, 1000.00)
	assert.InDeltaf(t, 100, wager, 0.1, "calculating wager for %v decimal odds with prob %v and multiplier %v", odds.decimalOdds, prob.percent, mult)
}

func TestOdds_Equals(t *testing.T) {
	odds1 := decimal(t, 1.5)
	odds2 := decimal(t, 1.5)
	odds3 := decimal(t, 2.0)

	assert.True(t, odds1.Equals(odds2))
	assert.False(t, odds2.Equals(odds3))
}

func TestOdds_Longer(t *testing.T) {
	odds1 := decimal(t, 1.5)
	odds2 := decimal(t, 1.5)
	odds3 := decimal(t, 2.0)
	assert.True(t, odds3.Longer(odds1))
	assert.False(t, odds2.Longer(odds1))
}

func TestOdds_Shorter(t *testing.T) {
	odds1 := decimal(t, 1.5)
	odds2 := decimal(t, 1.5)
	odds3 := decimal(t, 2.0)
	assert.True(t, odds1.Shorter(odds3))
	assert.False(t, odds1.Shorter(odds2))
}

func TestOdds_ExpectedValueProb(t *testing.T) {
	odds := american(t, -110.0)
	prob := NewProbabilityFromPercent(50.0)
	ev := odds.ExpectedValueProb(prob)
	assert.InDeltaf(t, -0.0455, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)

	odds = american(t, +180.0)
	prob = NewProbabilityFromPercent(30.0)
	ev = odds.ExpectedValueProb(prob)
	assert.InDeltaf(t, -0.16, ev, 0.001, "expected value of %v at %v% probability", odds.americanOdds, prob.percent)
}

func TestOdds_ExpectedValueOdds(t *testing.T) {
	odds := american(t, -110.0)
	trueOdds := american(t, +100.0)
	ev := odds.ExpectedValueOdds(trueOdds)
	assert.InDeltaf(t, -0.0455, ev, 0.001, "expected value of %v at %v% odds", odds.americanOdds, trueOdds.Decimal())

	odds = american(t, +180.0)
	trueOdds = american(t, +233.0)
	ev = odds.ExpectedValueOdds(trueOdds)
	assert.InDeltaf(t, -0.16, ev, 0.001, "expected value of %v at %v% odds", odds.americanOdds, trueOdds.Decimal())
}

func TestOdds_ArbTo(t *testing.T) {
	odds1 := american(t, -110.0)
	odds2 := american(t, -110.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = american(t, 100.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = american(t, 109.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = american(t, 110.0)
	assert.False(t, odds1.ArbTo(odds2))
	assert.False(t, odds2.ArbTo(odds1))

	odds2 = american(t, 111.0)
	assert.True(t, odds1.ArbTo(odds2))
	assert.True(t, odds2.ArbTo(odds1))
}

func TestOdds_ArbRoi(t *testing.T) {
	odds1 := decimal(t, 3.0)
	odds2 := decimal(t, 3.0)
	assert.Equal(t, 0.5, odds1.ArbRoi(odds2))

	odds1 = american(t, +150)
	odds2 = american(t, +150)
	assert.Equal(t, 0.25, odds1.ArbRoi(odds2))

	odds1 = american(t, -200)
	odds2 = american(t, -200)
	assert.Equal(t, -0.25, odds1.ArbRoi(odds2))
}

func TestOdds_ToString(t *testing.T) {
	odds := american(t, +200.0)
	assert.Equal(t, "+200.00", odds.ToString(American))
	assert.Equal(t, "3.00", odds.ToString(Decimal))

	assert.Equal(t, "33.33¢", odds.ToString(Cents))

	odds = american(t, -200.0)
	assert.Equal(t, "-200.00", odds.ToString(American))
	assert.Equal(t, "66.67¢", odds.ToString(Cents))

	odds, err := NewOddsFromCents(21.75)
	assert.Nil(t, err)
	assert.Equal(t, "21.75¢", odds.ToString(Cents))
}

func TestOddsFormat_ToString(t *testing.T) {
	assert.Equal(t, "american", American.ToString())
	assert.Equal(t, "decimal", Decimal.ToString())
	assert.Equal(t, "cents", Cents.ToString())
}

func TestOddsFormatFromString(t *testing.T) {
	format, err := OddsFormatFromString("american")
	assert.Equal(t, format, American)
	assert.Nil(t, err)

	format, err = OddsFormatFromString("decimal")
	assert.Equal(t, format, Decimal)
	assert.Nil(t, err)

	format, err = OddsFormatFromString("cents")
	assert.Equal(t, format, Cents)
	assert.Nil(t, err)

	format, err = OddsFormatFromString("foo")
	assert.Equal(t, format, Unknown)
	assert.NotNil(t, err)
}

func TestMarketWidth(t *testing.T) {
	odds1 := american(t, -141.0)
	odds2 := american(t, +123.0)
	assert.Equal(t, 18.0, MarketWidth(odds1, odds2))

	odds1 = american(t, -110.0)
	odds2 = american(t, -114.0)
	assert.Equal(t, 24.0, MarketWidth(odds1, odds2))

	odds1 = american(t, +150.0)
	odds2 = american(t, +137.0)
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

func dummyAverageOdds(t *testing.T) AverageOdds {
	ao := NewAverageOdds()
	ao.Accumulate(decimal(t, 3.0))
	ao.Accumulate(decimal(t, 5.0))
	ao.Accumulate(decimal(t, 7.0))
	return ao
}

func TestAverageOdds(t *testing.T) {
	ao := dummyAverageOdds(t)
	// 4.43661971844
	avg, err := ao.Average()
	assert.Nil(t, err)
	assert.InDeltaf(t, 4.4366, avg.decimalOdds, 0.00005, "averaging odds %v", ao)
}

func TestAverageOddsEmpty(t *testing.T) {
	ao := NewAverageOdds()
	_, err := ao.Average()
	assert.EqualError(t, err, "no odds to average")
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
				ao.AccumulateWeighted(2, decimal(t, 3.0))
				ao.AccumulateWeighted(1, decimal(t, 5.0))
			},
			want: 3.4615,
		},
		{
			// (1/7 + 3/3) / 4 = 0.28571, with Accumulate weighing one.
			name: "mixed with Accumulate",
			accumulate: func(ao *AverageOdds) {
				ao.Accumulate(decimal(t, 7.0))
				ao.AccumulateWeighted(3, decimal(t, 3.0))
			},
			want: 3.5,
		},
		{
			// (0.5/2 + 0.5/4) / 1 = 0.375, weights need not be whole.
			name: "fractional weights",
			accumulate: func(ao *AverageOdds) {
				ao.AccumulateWeighted(0.5, decimal(t, 2.0), decimal(t, 4.0))
			},
			want: 2.6667,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ao := NewAverageOdds()
			tt.accumulate(&ao)
			avg, err := ao.Average()
			assert.Nil(t, err)
			assert.InDeltaf(t, tt.want, avg.decimalOdds, 0.00005, "averaging odds %v", ao)
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
func sampleOdds1(t *testing.T) []Odds {
	return []Odds{decimal(t, 2.09), decimal(t, 3.59), decimal(t, 3.77)}
}

// sampleOdds2 returns the Odds from the tests at
// https://github.com/mberk/shin/blob/master/tests/test_shin.py.
func sampleOdds2(t *testing.T) []Odds {
	return []Odds{decimal(t, 2.6), decimal(t, 2.4), decimal(t, 4.3)}
}

// tenRunners returns a field of ten runners at a 20% margin, as a race or a
// futures market has. The old fixed point solver never converged on it for
// Shin or the logarithmic method.
func tenRunners(t *testing.T) []Odds {
	return decimalOdds(t, 11.14, 5.97, 52.76, 10.9, 5.76, 5.11, 10.44, 10.55, 11.02, 5.51)
}

// underround returns a two way market whose implied probabilities sum to
// 95.6%, which the solvers must push the other way.
func underround(t *testing.T) []Odds {
	return decimalOdds(t, 1.8, 2.5)
}

func decimalOdds(t *testing.T, prices ...float64) []Odds {
	t.Helper()
	var odds []Odds
	for _, p := range prices {
		odds = append(odds, decimal(t, p))
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

func TestVig(t *testing.T) {
	// -110/-110: an overround of 2 * 110/210 = 1.047619, a margin of 4.76% and a vig of 4.55%.
	assert.InDelta(t, 0.0476190/1.0476190, Vig(american(t, -110), american(t, -110)), 1e-6)
	assert.Equal(t, 0.0455, round(Vig(american(t, -110), american(t, -110)), 4))
	// A fair market has none, and one priced above fair, an arb, has a negative vig.
	assert.InDelta(t, 0.0, Vig(decimal(t, 2.0), decimal(t, 2.0)), 1e-12)
	assert.Less(t, Vig(decimal(t, 2.1), decimal(t, 2.1)), 0.0)
	// Three outcomes: 1/2 + 1/3.5 + 1/4.2 = 1.02381, a vig of 2.33%.
	assert.InDelta(t, 0.02381/1.02381, Vig(decimalOdds(t, 2.0, 3.5, 4.2)...), 1e-5)
}

func TestEqualMarginOdds(t *testing.T) {
	trueOdds, err := EqualMarginOdds()
	assert.NotNil(t, err)

	trueOdds, err = EqualMarginOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = EqualMarginOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1365, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6700, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8540, round(trueOdds[2].decimalOdds, 4))
}

func TestMPTOdds(t *testing.T) {
	trueOdds, err := MPTOdds()
	assert.NotNil(t, err)

	trueOdds, err = MPTOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = MPTOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1229, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6883, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8786, round(trueOdds[2].decimalOdds, 4))

	// It takes margin / n from each implied probability, so a long enough price
	// in a wide enough book has more taken than it has. An 18.3% margin over
	// three prices takes 6.1% from the +1900's 5%. The others then sum to over
	// one, so none of them is fair either.
	trueOdds, err = MPTOdds(american(t, -400), american(t, 200), american(t, 1900))
	assert.Nil(t, trueOdds)
	assert.NotNil(t, err)

	// A 5% margin takes 1.7% from the same longshot.
	trueOdds, err = MPTOdds(american(t, -300), american(t, 300), american(t, 1900))
	assert.Nil(t, err)
	assertProbs(t, []float64{0.7333333, 0.2333333, 0.0333333}, trueOdds)
}

func TestShinOdds(t *testing.T) {
	trueOdds, err := ShinOdds()
	assert.NotNil(t, err)

	trueOdds, err = ShinOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = ShinOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1264, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6836, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8723, round(trueOdds[2].decimalOdds, 4))

	// mberk/shin's test gives these to seven places.
	trueOdds, err = ShinOdds(sampleOdds2(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 0.3729941, round(trueOdds[0].ImpliedProb().decimal, 7))
	assert.Equal(t, 0.4047794, round(trueOdds[1].ImpliedProb().decimal, 7))
	assert.Equal(t, 0.2222265, round(trueOdds[2].ImpliedProb().decimal, 7))

	trueOdds, err = ShinOdds(tenRunners(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0719737, 0.1433818, 0.0093293, 0.0737813, 0.1490059, 0.1693519, 0.0774805, 0.0765663, 0.0728675, 0.1562619}, trueOdds)

	trueOdds, err = ShinOdds(underround(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5777778, 0.4222222}, trueOdds)
}

func TestOddsRatioOdds(t *testing.T) {
	trueOdds, err := OddsRatioOdds()
	assert.NotNil(t, err)

	trueOdds, err = OddsRatioOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = OddsRatioOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1285, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6814, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8678, round(trueOdds[2].decimalOdds, 4))

	trueOdds, err = OddsRatioOdds(tenRunners(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0740145, 0.1402120, 0.0154173, 0.0756730, 0.1454978, 0.1647182, 0.0790689, 0.0782294, 0.0748345, 0.1523345}, trueOdds)

	trueOdds, err = OddsRatioOdds(underround(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5779355, 0.4220645}, trueOdds)

	// A price of 1.0 has an implied probability of one at every c, so no c
	// brings the sum down to one. The constructors reject 1.0, so it's made
	// directly: the solvers still guard against it, as for a zero Odds.
	_, err = OddsRatioOdds(Odds{decimalOdds: 1.0, netFractional: 0}, decimal(t, 2.0))
	assert.EqualError(t, err, "no true odds for decimal odds [1 2]")
}

func TestLogarithmicOdds(t *testing.T) {
	trueOdds, err := LogarithmicOdds()
	assert.NotNil(t, err)

	trueOdds, err = LogarithmicOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = LogarithmicOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1230, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6888, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8778, round(trueOdds[2].decimalOdds, 4))

	trueOdds, err = LogarithmicOdds(tenRunners(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0720491, 0.1423145, 0.0132009, 0.0737819, 0.1479856, 0.1686414, 0.0773364, 0.0764569, 0.0729057, 0.1553274}, trueOdds)

	trueOdds, err = LogarithmicOdds(underround(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5763815, 0.4236185}, trueOdds)

	// A price of 1.0 has an implied probability of one at every c, so no c
	// brings the sum down to one. The constructors reject 1.0, so it's made
	// directly: the solvers still guard against it, as for a zero Odds.
	_, err = LogarithmicOdds(Odds{decimalOdds: 1.0, netFractional: 0}, decimal(t, 2.0))
	assert.EqualError(t, err, "no true odds for decimal odds [1 2]")
}

func TestProbitOdds(t *testing.T) {
	trueOdds, err := ProbitOdds()
	assert.NotNil(t, err)

	trueOdds, err = ProbitOdds(decimal(t, 2.0))
	assert.NotNil(t, err)

	trueOdds, err = ProbitOdds(sampleOdds1(t)...)
	assert.Nil(t, err)
	assert.Equal(t, 2.1273, round(trueOdds[0].decimalOdds, 4))
	assert.Equal(t, 3.6829, round(trueOdds[1].decimalOdds, 4))
	assert.Equal(t, 3.8700, round(trueOdds[2].decimalOdds, 4))

	trueOdds, err = ProbitOdds(tenRunners(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.0731016, 0.1411973, 0.0143810, 0.0747955, 0.1466588, 0.1665270, 0.0782668, 0.0774083, 0.0739390, 0.1537247}, trueOdds)

	trueOdds, err = ProbitOdds(underround(t)...)
	assert.Nil(t, err)
	assertProbs(t, []float64{0.5779018, 0.4220982}, trueOdds)

	// A price of 1.0 has an implied probability of one at every c, so no c
	// brings the sum down to one. The constructors reject 1.0, so it's made
	// directly: the solvers still guard against it, as for a zero Odds.
	_, err = ProbitOdds(Odds{decimalOdds: 1.0, netFractional: 0}, decimal(t, 2.0))
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
			odds := decimal(t, tt.decimalOdds)
			trueOdds := decimal(t, 1/tt.prob)
			assert.InDelta(t, tt.want, odds.Meg(trueOdds), 1e-5)
		})
	}
}
