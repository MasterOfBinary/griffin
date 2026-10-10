# Setup

Griffin is a local paper-trading lab for three liquid majors: SOL, BTC, and ETH, quoted in USDC. It replays a dataset, records signals, and optionally records paper fills. It does not hold key material, does not sign, does not submit transactions, does not call an execution venue, and does not buy assets.

The GitHub default branch is `master`. Use that branch. There is no `main` branch.

## Requirements

- Go 1.22 or newer
- A shell on the machine that will run the lab

Not used: Docker, a database server, a frontend toolchain, or a remote market connection.

## Build and test

From the repository root:

```sh
make test
make build
make demo
```

`make demo` is `./bin/griffin demo --once`. It reads `fixtures/synthetic/majors.json`, stores a finished run under `var/griffin`, and exits. That directory is gitignored.

To open the UI after the same replay:

```sh
./bin/griffin demo
```

The process listens on `127.0.0.1:8787` and refuses any other address, including `localhost` and `0.0.0.0`. Stop it with Ctrl-C.

## Modes

| Mode | CLI | What is stored |
| --- | --- | --- |
| `demo` | `griffin demo` or `griffin run --mode demo` | Signals and paper fills |
| `signal_only` | `griffin run --mode signal_only` | Signals only; cash stays at the starting balance |
| `strict` | `griffin run --mode strict` | Paper fills only when every bar passes the spread and liquidity gates. A failing bar is stored as a finished halted run with no fills |

The same three labels are printed by the CLI, stored on the run record, and shown in the UI.

## Run ids

```text
run_id = mode + "-v" + config_version + "-" + hex(first 8 bytes of SHA-256(mode + "\n" + config_version + "\n" + dataset_hash))
```

`dataset_hash` is the SHA-256 of the dataset file bytes. The same mode, configuration version, and dataset always produce the same id. Griffin refuses to store that id again after the run is finished. Change the mode, bump `config_version` when settings change, or point `dataset` at a new file to choose a new id.

## Config

Copy `config.example.json` if you want a local file. Relative dataset paths are resolved from the working directory and then from the config file's directory. Run commands from the repository root.

`fixtures/recorded/` is gitignored. A later recording can use the same JSON shape as `fixtures/synthetic/majors.json` and be selected with `dataset`. The demo path stays on the synthetic file so it runs offline.

## Store

Finished runs are written to `data_dir/runs.json`. The file is ordinary JSON. There is no database server. Griffin refuses to write a record that matches its key-material detector, and it refuses to load a store file that matches.
