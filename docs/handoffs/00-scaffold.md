# Handoff: scaffold

Griffin is the liquid-majors paper lab. The checked-in path is a synthetic replay plus a loopback UI. Recorded datasets can be added later under `fixtures/recorded/` (gitignored) by pointing `dataset` at a file with the same JSON shape.

## Stay inside these bounds

- Universe changes are code changes in `internal/market`. Do not accept long-tail symbols from config.
- Do not add a signer, a submission path, an execution-venue client, or a buy path. `internal/forbid` fails `go test` when those paths show up in the tree. The only skipped source file is `internal/forbid/denylist.go`, because it has to name them.
- Do not add Docker, a database server, or a frontend build. Banned filenames include a container file and `package.json`.
- Key material stays out of fixtures, the JSON store, logs, errors, and the UI. `internal/safety` redacts detector matches, and the store refuses to write or load them.
- Run ids stay a pure function of mode, `config_version`, and the dataset hash. Do not add a silent overwrite of a finished id.

## Modes

`demo`, `signal_only`, and `strict` are the only labels. They are parsed in `internal/safety`, required by the CLI, stored on the run, and rendered in `internal/web`.

## Next research step

Replace or extend the synthetic fixture with a local recording of the same three majors, bump `config_version` if the strategy parameters change, and compare the stored signal list. Keep fills inside `internal/sim`.
