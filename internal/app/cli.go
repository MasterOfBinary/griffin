package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MasterOfBinary/griffin/internal/store"
	"github.com/MasterOfBinary/griffin/internal/web"
)

const usage = `Griffin is a paper-trading proving ground for liquid majors.

Usage:
  griffin demo [--config path] [--once] [--listen host:port] [--data-dir dir]
  griffin run --mode demo|signal_only|strict [--config path] [--once] [--listen host:port] [--data-dir dir]
  griffin serve [--config path] [--listen host:port] [--data-dir dir]

Modes:
  demo         replay the dataset and record paper fills
  signal_only  record signals and do not record fills
  strict       record paper fills only when every bar passes the safety gates

Run ids are derived from mode, configuration version, dataset hash, and the research settings.
griffin run requires --mode. A finished id is not recomputed.
griffin demo serves the UI for a finished demo run instead of refusing it.
griffin demo --once refuses a second execution of the same id.
The UI binds to a loopback address only. Names are refused.
The GitHub default branch for this repository is master.
`

type options struct {
	configPath string
	once       bool
	listen     string
	dataDir    string
	mode       string
}

// Execute runs a griffin command. stdout receives the summary and the listen line.
func Execute(ctx context.Context, args []string, stdout io.Writer) error {
	if stdout == nil {
		stdout = os.Stdout
	}
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return errors.New("missing command")
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	case "demo":
		return runLabelled(ctx, args[1:], stdout, "demo", true)
	case "run":
		return runLabelled(ctx, args[1:], stdout, "", true)
	case "serve":
		return runServe(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage)
	}
}

func runLabelled(ctx context.Context, args []string, stdout io.Writer, forceMode string, allowOnce bool) error {
	opt, err := parseFlags("griffin", args, forceMode == "", allowOnce)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return nil
		}
		return err
	}
	if forceMode == "" && opt.mode == "" {
		return errors.New("griffin run requires --mode demo, signal_only, or strict")
	}
	cfg, err := loadAndApply(opt, forceMode)
	if err != nil {
		return err
	}
	run, err := RunOnce(cfg)
	if err != nil {
		if opt.once || !errors.Is(err, store.ErrRunFinished) {
			return err
		}
		existing, gerr := finishedRun(cfg)
		if gerr != nil {
			return err
		}
		fmt.Fprint(stdout, Format(existing))
		fmt.Fprintln(stdout, "serve: existing finished run")
		return serveStore(ctx, cfg, stdout)
	}
	fmt.Fprint(stdout, Format(run))
	if opt.once {
		fmt.Fprintln(stdout, "serve: skipped")
		return nil
	}
	return serveStore(ctx, cfg, stdout)
}

func runServe(ctx context.Context, args []string, stdout io.Writer) error {
	opt, err := parseFlags("griffin", args, false, false)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return nil
		}
		return err
	}
	cfg, err := loadAndApply(opt, "")
	if err != nil {
		return err
	}
	return serveStore(ctx, cfg, stdout)
}

func serveStore(ctx context.Context, cfg Config, stdout io.Writer) error {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	ln, err := web.Listen(cfg.Listen)
	if err != nil {
		return err
	}
	bound := ln.Addr().String()
	fmt.Fprintf(stdout, "listening: http://%s\n", bound)
	return web.Serve(ctx, ln, web.New(st, bound))
}

func loadAndApply(opt options, forceMode string) (Config, error) {
	cfg, err := LoadConfig(opt.configPath)
	if err != nil {
		return Config{}, err
	}
	if opt.listen != "" {
		cfg.Listen = opt.listen
	}
	if opt.dataDir != "" {
		cfg.DataDir = opt.dataDir
	}
	if forceMode != "" {
		cfg.Mode = forceMode
	} else if opt.mode != "" {
		cfg.Mode = opt.mode
	}
	cfg.Dataset = resolveExisting(opt.configPath, cfg.Dataset)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func resolveExisting(configPath, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	alt := filepath.Join(filepath.Dir(configPath), p)
	if _, err := os.Stat(alt); err == nil {
		return alt
	}
	return p
}

func parseFlags(name string, args []string, withMode, withOnce bool) (options, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var opt options
	fs.StringVar(&opt.configPath, "config", "config.example.json", "path to config JSON")
	fs.StringVar(&opt.listen, "listen", "", "loopback host:port override")
	fs.StringVar(&opt.dataDir, "data-dir", "", "data directory override")
	if withOnce {
		fs.BoolVar(&opt.once, "once", false, "store the run and skip the UI")
	}
	if withMode {
		fs.StringVar(&opt.mode, "mode", "", "demo, signal_only, or strict")
	}
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if extra := fs.Args(); len(extra) > 0 {
		return options{}, fmt.Errorf("unexpected argument %q", extra[0])
	}
	return opt, nil
}
