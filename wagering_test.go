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

func TestProbability(t *testing.T) {
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
		assert.InDeltaf(t, ep.prob, ep.odds.probability(), 0.01, "converting decimal %v", ep.odds.decimalOdds)
	}
}