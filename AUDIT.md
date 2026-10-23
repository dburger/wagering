# Audit of wagering

2026-10-23, at c9adb64. Scope: `wagering.go`, `wagering_test.go`, `README.md` and `go.mod`.

Each behavior below was confirmed in the code, and the ones marked "probe" by a throwaway test
run inside the package (not committed). Callers were checked too: odds, plays and bet check every
constructor's error in production code, so none of these gives a wrong number in those tools
today. Most findings are gaps in the library's contract, and a few are traps for a future caller.

## Bugs and traps

| # | Problem | Evidence | Suggested fix |
|---|---|---|---|
| 1 | **The zero `Odds{}` says to stake the whole bankroll.** `KellyFraction` returns 1 and `Meg` returns NaN. `ImpliedProb` returned +Inf; since #2's fix it returns 0, which makes `ArbTo` report an arbitrage against every valid price (probe: against 1.5). `Odds` has no validity check, so a caller who ignores a constructor's error gets confident nonsense. | probe: `KellyFraction` = 1 | **Fixed 2026-10-23** (branch `invalid-odds`): `Valid()` reports whether an `Odds` is a price (decimal above 1). For one that isn't, `KellyFraction`, `KellyStake`, both expected value methods, `Meg` and `ArbRoi` return 0, `ArbTo` false and `Vig` NaN; the devig functions return an error, and `Average` does if one was accumulated. |
| 2 | **`Probability` accepts anything** (1.7, -0.2, NaN) and has **no exported accessors**, so callers outside the package can neither check it nor read it back. | probe: `NewProbabilityFromDecimal(1.7)` holds 170% | **Fixed 2026-10-23** (branch `probability-validation`): `NewProbabilityFromDecimal` and `NewProbabilityFromPercent` return an error outside 0 to 1 (0 to 100 percent), NaN and infinities included, and `Decimal()` and `Percent()` read a `Probability` back. |
| 3 | **`AccumulateWeighted` accepts negative or NaN weights.** A weight of -1 averages without complaint, and mixing signs gives garbage. plays checks its weights first; the library doesn't. | probe | **Fixed 2026-10-23** (branch `average-weights`): a negative, NaN or infinite weight makes `Average` an error naming it. A weight of 0 still counts its odds for nothing, rather than being rejected as suggested, since plays treats sharps whose weights are all 0 as giving no true odds. `AverageOdds` keeps the first problem, an invalid weight or `Odds`, and `Average` returns it. |
| 4 | **`ToString(Unknown)` panics,** yet `Unknown` is what `OddsFormatFromString` returns with its error. | probe | **Fixed 2026-10-23** (branch `tostring-fallback`): any format but american and cents, `Unknown` included, gives decimal, as a price in the wrong format is still the right price. `ToString` is documented, and the library has no `panic` left. |
| 5 | **Cents' american odds aren't exact.** 544 of the 998 tenth-of-a-cent prices miss the nearest float64, by a relative error of at most 7.65e-14. Harmless in practice, but it's the one constructor that breaks the rule stated above the constructors, which american and decimal follow (and test). | probe | **Fixed 2026-10-23** (branch `cents-exact`): american and net fractional odds are computed exactly with `math/big` rationals from cents' own value and rounded once, as no float64 formula managed (computing them directly in float64 still missed 24% to 31% of prices, and an FMA-corrected version 10%). `TestNewOddsFromCentsExact` checks every tenth of a cent and 20,000 random prices; master's code fails 12,956 of them. |

## API design

| # | Problem | Suggested fix |
|---|---|---|
| 6 | **The `OddsFormat` values are exported variables,** so a caller can reassign `wagering.American` (the TODO above them). They also print as `{bogus}` in errors (probe: `NewOdds`' "unknown odds format: {bogus}"), since `OddsFormat` has no `String()`. | `type OddsFormat string` with constants, `const American OddsFormat = "american"`: the same names for callers, immutable, and they print as their slug. odds, plays and bet would compile unchanged, as none calls `OddsFormat.ToString()`. |
| 7 | **Fractional odds are half there.** `netFractional` is stored in every `Odds`, but there's no Fractional format, constructor or accessor, and only `Meg` reads it. oddspunk supports fractional odds. | Finish it (format, constructor, `ToString`), or drop the field. |
| 8 | **Arbs are two-way only.** `ArbTo` and `ArbRoi` take exactly two prices, and `ArbTo` carries a commented-out old implementation. `ArbRoi` equals 1/overround - 1 (probe: both 0.037349), which works for any number of outcomes. | Variadic `ArbRoi(odds ...Odds)`, and an `ArbTo` on the overround; delete the dead code. |
| 9 | **No `NewOddsFromProbability`.** `Average` (with a TODO asking for it), `transOdds` and `MPTOdds` build odds from a probability by hand with `NewOddsFromDecimal(1/p)`, and plays does the same. | **Fixed 2026-10-23** (branch `odds-from-probability`): `NewOddsFromProbability` takes a `Probability` and rejects 0 and 1, which aren't prices. Like cents (#5), which now shares its code (`oddsFromProb`), every format is computed exactly and rounded once. `Average`, `transOdds` and `MPTOdds` use it, which removes `Average`'s TODO. plays' output on a real props snapshot is identical, 60,305 lines, at about 12% more time (0.89s to 1.01s a run). |
| 10 | **Units are inconsistent.** `Meg` returns basis points, while `ExpectedValueProb`, `ExpectedValueOdds`, `Vig` and Kelly return fractions; plays divides `Meg` by 10000. | Fractions throughout, with callers converting for display. A breaking change for plays. |
| 11 | **`MarketWidth` has no caller** in odds, plays or bet. It uses the american "cents" convention, plus one of its own when both sides are plus odds ("My own concoction"). | Keep it, documented, or remove it. |

## Docs

12. **The package example doesn't compile.** It passes a `float64` likelihood to `KellyStake`,
    which takes a `Probability`. Its note on precision also leaves out cents.
13. **`ExpectedValueProb` and `ExpectedValueOdds` say "percent increase"** but return a fraction
    (probe: 0.05 for a 5% edge).
14. **The devig functions don't give their common names.** `EqualMarginOdds` is the
    multiplicative method and `MPTOdds` the additive one; plays maps the names itself. A line each
    would help.
15. **The README is three lines.** A short list of the odds formats, the devig methods and the
    units would cover it.

## Tests and housekeeping

16. **Test helpers share names with library code.** `decimalOdds(t, prices...)` in
    `wagering_test.go` is hidden by locals in `NewOddsFromAmerican` and `NewOddsFromCents` and by
    `NewOddsFromDecimal`'s parameter; `decimal(t, decimalOdds)` by `NewProbabilityFromDecimal`'s
    parameter. Only test builds see the helpers, and the library's names match the
    `Odds.decimalOdds` field, so rename the helpers, e.g. `oddsFromDecimals` and `fromDecimal`.
    Neither `go vet` nor the x/tools `shadow` analyzer (even with `-strict`) reports a local that
    hides a package-level function, so this needs a reader's eye.
17. Smaller things:
    - Five devig functions return "need at least two odds" without saying which one failed.
    - `go.mod` says `go 1.20`, while odds and plays are on Go 1.26.
    - `solveForC` recomputes each price's implied probability on every bisection step. That's
      cheap at today's scale, so it's low priority.

## Suggested order

1. **#1 to #5 and #9:** they close the traps without breaking any caller. #1 to #5 and #9 are fixed.
2. **#6, #12 and #13:** cheap, and #6 keeps the identifiers callers use.
3. **#7, #8, #10 and #11:** API choices to decide first, since #10 changes what plays receives.
