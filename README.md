# wagering

A Go library for odds and the math of betting on them: converting between odds formats, removing
the vig from a market to get its true odds, and sizing and judging a bet against them. odds, plays
and bet use it.

```go
odds, err := wagering.NewOddsFromAmerican(-110)
if err != nil {
	return err
}
likelihood, err := wagering.NewProbabilityFromDecimal(0.6)
if err != nil {
	return err
}
stake := odds.KellyStake(likelihood, 0.3, 10000) // 480, at 0.3 Kelly
```

## Odds

An `Odds` holds a price in every format at once:

| Format | Constructor | Example |
|---|---|---|
| american | `NewOddsFromAmerican` | -110, +150 |
| decimal | `NewOddsFromDecimal` | 1.91, 2.50 |
| cents, a prediction market's price of a contract paying 100 | `NewOddsFromCents` | 52.38, 40 |
| probability, the chance of winning it implies | `NewOddsFromProbability` | a `Probability` of 0.4 |

`NewOdds(price, format)` takes the format as an `OddsFormat` (`American`, `Decimal`, `Cents`), and
`OddsFormatFromString` turns a name such as `"cents"` into one. `ToString(format)` writes a price
as `-110.00`, `1.91` or `52.38¢`.

- **Exact.** Each constructor computes every format directly from the value it's given, so each
  is the float64 nearest its exact value: always for decimal odds, cents and probabilities, and
  for american odds when they're whole numbers. The tests check this against exact rationals.
- **Validated.** A constructor rejects what isn't a price (american between -100 and +100,
  decimal 1 or less, cents outside 0 to 100, a probability of 0 or 1, NaN and infinities) with an
  error. The zero `Odds` isn't a price either: `Valid()` says so, and the calculations return 0
  for it, `ArbTo` false and `Vig` NaN, rather than a plausible wrong answer.
- **Probabilities** are a `Probability`, from `NewProbabilityFromDecimal` (0 to 1) or
  `NewProbabilityFromPercent` (0 to 100), so a 0.6 is never mistaken for 60%. `Decimal()` and
  `Percent()` read one back.

## Calculations

| Method | Returns |
|---|---|
| `ImpliedProb()` | the break-even probability, 1 over the decimal odds |
| `ExpectedValueProb(p)`, `ExpectedValueOdds(trueOdds)` | the expected gain per unit staked, as a fraction |
| `KellyFraction(p, mult)`, `KellyStake(p, mult, bankroll)` | the Kelly bet, at a multiplier such as 0.25 |
| `Meg(trueOdds)` | the maximum expected growth, the bankroll's growth per bet at full Kelly, in **basis points** |
| `ArbTo(other)`, `ArbRoi(other)` | whether two prices on opposite sides are an arbitrage, and its return |
| `Vig(odds...)` | a market's vig, its margin over its overround: -110/-110 is 4.55% |
| `NewAverageOdds()`, `Accumulate`, `AccumulateWeighted`, `Average` | an average of odds by their implied probabilities, optionally weighted |

Results are fractions, 0.05 for 5%, except `Meg`, which is in basis points.

## Devigging

Each method takes a market, one `Odds` per outcome, and returns its true odds, with the vig
removed. They differ in how they share the vig out among the outcomes:

| Function | Common name | How |
|---|---|---|
| `EqualMarginOdds` | multiplicative, normalization | divides each implied probability by the overround |
| `MPTOdds` | additive, margin proportional to odds | subtracts an equal share of the margin from each |
| `LogarithmicOdds` | power | raises each implied probability to the same power |
| `ShinOdds` | Shin | Shin's model of the vig as protection against insiders |
| `OddsRatioOdds` | odds ratio | divides each outcome's odds, p / (1 - p), by the same factor |
| `ProbitOdds` | probit | shifts each outcome by the same amount in probit space |

The last four solve for their parameter by bisection. A market needs two or more outcomes, all
`Valid`; an error names the function. On two-way markets Shin is the same as additive.

## Development

```
make test        # go test ./...
```

`AUDIT.md` records an audit of the library (2026-10-23) and what has been fixed since.
