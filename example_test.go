package wagering_test

import (
	"fmt"

	"github.com/dburger/wagering"
)

// The package documentation's example: a stake at 0.3 Kelly on a 60% chance at -110. Written as
// an example so that go test compiles and runs it, and it can't drift from the API again.
func ExampleOdds_KellyStake() {
	odds, err := wagering.NewOddsFromAmerican(-110.0)
	if err != nil {
		fmt.Println(err)
		return
	}
	likelihood, err := wagering.NewProbabilityFromDecimal(0.6)
	if err != nil {
		fmt.Println(err)
		return
	}
	wager := odds.KellyStake(likelihood, 0.3, 10000.0)
	fmt.Printf("%.2f\n", wager)
	// Output: 480.00
}

// An expected value is a fraction of the stake: +110 when the true chance is 50% gains 5%.
func ExampleOdds_ExpectedValueProb() {
	odds, _ := wagering.NewOddsFromAmerican(110)
	half, _ := wagering.NewProbabilityFromDecimal(0.5)
	fmt.Printf("%.4f\n", odds.ExpectedValueProb(half))
	// Output: 0.0500
}
