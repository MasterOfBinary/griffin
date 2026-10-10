# Strategy 1: school-griffin-1

School-griffin-1 is a research signal on the closed liquid-major universe (SOL, BTC, ETH). It is a dual exponential moving average cross on bar closes.

## Parameters

| Name | Example | Meaning |
| --- | --- | --- |
| `fast` | 8 | Fast average period |
| `slow` | 21 | Slow average period. Must be greater than `fast` |
| `max_position_pct` | 0.2 | Fraction of current paper cash allocated on an entry |

Each average is seeded with a simple average of its first window. A bar becomes eligible once both averages exist. The first eligible bar only establishes the prior spread. Later bars emit:

- `enter_long` when the fast average crosses from at or below the slow average to above it
- `exit` when the fast average crosses from at or above the slow average to below it

Holds are not stored. The UI treats every other bar as a hold.

## Paper fills

A signal is not filled on the signal bar. In `demo`, and in `strict` when the book passes the gates, the fill happens at the next bar's open.

- Buys pay the open plus slippage and a fee, then receive a quantity sized from `max_position_pct` of current paper cash, capped by `safety.max_notional`.
- Sells receive the open minus slippage, minus a fee, and close the paper quantity.

`signal_only` keeps the signals and does not record fills.

`strict` checks every bar's `spread_bps` and `liquidity_usd` before any fill. The first failure finishes the run as halted and records no fills. That finished id cannot be reused. Repairing the dataset changes its hash and therefore its run id.

## What this strategy does not do

It does not size from borrowed funds, it does not short, and it does not look outside the three majors. Parameter changes belong in a new `config_version` so the run id changes with the research claim.
