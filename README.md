# griffin

Griffin is a research and paper-trading lab for liquid majors (SOL, BTC, ETH, quoted in USDC). It replays a local dataset, labels each run as `demo`, `signal_only`, or `strict`, and can show the stored result on a loopback-only UI.

This is a proving ground. Griffin does not hold key material, does not sign, does not submit transactions, does not call an execution venue, and does not buy assets.

The GitHub default branch is **`master`**.

## Run the offline demo

Requires Go 1.22 or newer. From the repository root:

```sh
make test
make build
./bin/griffin demo
```

`./bin/griffin demo` replays `fixtures/synthetic/majors.json` and listens on `127.0.0.1:8787`. Any other bind address is refused. `make demo` is the same replay with `--once`, which stores the run and skips the UI.

A second demo with the same mode, `config_version`, and dataset exits with an error. Bump `config_version`, change mode, or change the dataset file to choose a new run id.

```sh
./bin/griffin run --mode signal_only --once
./bin/griffin run --mode strict --once
./bin/griffin serve
```

Finished runs are JSON files under `var/griffin/` (gitignored). There is no database server, no Docker, and no frontend build.

## Layout

```text
cmd/griffin/          CLI
internal/market/      closed liquid-major universe
internal/candles/     local OHLCV fixtures
internal/strategy/    school-griffin-1 signal
internal/sim/         paper fills
internal/safety/      modes, loopback bind, key redaction
internal/store/       finished-run JSON store
internal/web/         loopback UI
internal/forbid/      repository scan that fails the build
internal/app/         config, run pipeline, CLI
docs/setup.md
docs/strategy-1.md
docs/handoffs/
fixtures/synthetic/   offline demo dataset
fixtures/recorded/    gitignored local recordings
```

See [docs/setup.md](docs/setup.md) and [docs/strategy-1.md](docs/strategy-1.md).
