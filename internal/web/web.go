// Package web serves the paper-lab UI. It binds loopback addresses only.
package web

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"

	"github.com/MasterOfBinary/griffin/internal/safety"
	"github.com/MasterOfBinary/griffin/internal/store"
)

// Serve listens on addr and serves h until ctx is cancelled.
// A non-loopback addr is refused before the socket opens.
func Serve(ctx context.Context, addr string, h http.Handler) error {
	if err := safety.ValidateListen(addr); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

// New returns the lab UI over an existing store.
func New(st *store.Store, listen string) http.Handler {
	mux := http.NewServeMux()
	h := &handler{store: st, listen: listen, tmpl: mustTemplate()}
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /runs/{id}", h.detail)
	mux.HandleFunc("GET /", h.index)
	return mux
}

type handler struct {
	store  *store.Store
	listen string
	tmpl   *template.Template
}

type modeView struct {
	Name   string
	Active bool
}

type page struct {
	Listen string
	Modes  []modeView
	Runs   []store.Run
	Run    *store.Run
	Found  bool
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "ok\nlisten=%s\n", h.listen)
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	h.render(w, "index", page{
		Listen: h.listen,
		Modes:  modeViews(""),
		Runs:   h.store.List(),
	})
}

func (h *handler) detail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, ok := h.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.render(w, "detail", page{
		Listen: h.listen,
		Modes:  modeViews(run.Mode),
		Run:    &run,
		Found:  true,
	})
}

func (h *handler) render(w http.ResponseWriter, name string, p page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, p); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func modeViews(active string) []modeView {
	out := make([]modeView, len(safety.Modes))
	for i, mode := range safety.Modes {
		name := string(mode)
		out[i] = modeView{Name: name, Active: name == active}
	}
	return out
}

func mustTemplate() *template.Template {
	return template.Must(template.New("griffin").Funcs(template.FuncMap{
		"redact": safety.Redact,
		"f2":     func(v float64) string { return fmt.Sprintf("%.2f", v) },
		"f4":     func(v float64) string { return fmt.Sprintf("%.4f", v) },
	}).Parse(indexHTML + detailHTML))
}

const indexHTML = `
{{define "index"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Griffin paper lab</title>
  <style>
    body { font-family: Georgia, "Iowan Old Style", serif; margin: 2rem auto; max-width: 960px; color: #1c1915; background: #f6f1e7; padding: 0 1rem; }
    h1 { font-weight: 500; margin-bottom: 0.25rem; }
    .banner, .modes, table { margin: 1rem 0; }
    .banner { padding: 0.75rem 1rem; background: #efe4cf; border: 1px solid #d9c7a3; }
    .modes { display: flex; gap: 0.5rem; list-style: none; padding: 0; }
    .modes li { padding: 0.15rem 0.55rem; border: 1px solid #1c1915; font-family: ui-monospace, monospace; }
    .modes li.active { background: #1c1915; color: #f6f1e7; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border-bottom: 1px solid #d9c7a3; text-align: left; padding: 0.4rem; vertical-align: top; }
    a { color: #1c1915; }
    code { font-family: ui-monospace, monospace; }
  </style>
</head>
<body>
  <h1>Griffin</h1>
  <p>Paper-trading proving ground for liquid majors.</p>
  <div class="banner">Loopback UI only. Bound to <code>{{.Listen}}</code>. Research records stay on this machine.</div>
  <ul class="modes">
    {{range .Modes}}<li class="{{if .Active}}active{{end}}">{{.Name}}</li>{{end}}
  </ul>
  {{if .Runs}}
  <table>
    <thead><tr><th>Run</th><th>Mode</th><th>Config</th><th>Signals</th><th>Fills</th><th>Status</th></tr></thead>
    <tbody>
    {{range .Runs}}
      <tr>
        <td><a href="/runs/{{.ID}}"><code>{{.ID}}</code></a></td>
        <td class="mode">{{.Mode}}</td>
        <td>{{.ConfigVersion}}</td>
        <td>{{len .Signals}}</td>
        <td>{{len .Fills}}</td>
        <td>{{.Status}}{{if .Halted}} (halted){{end}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{else}}
  <p>No finished runs yet.</p>
  {{end}}
</body>
</html>
{{end}}
`

const detailHTML = `
{{define "detail"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Griffin {{.Run.Mode}}</title>
  <style>
    body { font-family: Georgia, "Iowan Old Style", serif; margin: 2rem auto; max-width: 960px; color: #1c1915; background: #f6f1e7; padding: 0 1rem; }
    h1 { font-weight: 500; }
    .banner, .modes, table { margin: 1rem 0; }
    .banner { padding: 0.75rem 1rem; background: #efe4cf; border: 1px solid #d9c7a3; }
    .modes { display: flex; gap: 0.5rem; list-style: none; padding: 0; }
    .modes li { padding: 0.15rem 0.55rem; border: 1px solid #1c1915; font-family: ui-monospace, monospace; }
    .modes li.active { background: #1c1915; color: #f6f1e7; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border-bottom: 1px solid #d9c7a3; text-align: left; padding: 0.4rem; vertical-align: top; }
    a { color: #1c1915; }
    code, .mono { font-family: ui-monospace, monospace; }
  </style>
</head>
<body>
  <p><a href="/">All runs</a></p>
  <h1>{{.Run.StrategyName}}</h1>
  <div class="banner">Mode <strong class="mode">{{.Run.Mode}}</strong>. Loopback UI bound to <code>{{.Listen}}</code>.</div>
  <ul class="modes">
    {{range .Modes}}<li class="{{if .Active}}active{{end}}">{{.Name}}</li>{{end}}
  </ul>
  <p class="mono">run {{.Run.ID}}<br>config {{.Run.ConfigVersion}}<br>dataset {{.Run.DatasetName}}<br>hash {{.Run.DatasetHash}}</p>
  {{if .Run.Halted}}<p class="banner">Halted: {{redact .Run.HaltReason}}</p>{{end}}
  <p>Cash {{f2 .Run.Cash}} · equity {{f2 .Run.Equity}} · started from {{f2 .Run.StartingCash}}</p>
  <p>Bars without a signal row are holds.</p>
  <h2>Signals</h2>
  {{if .Run.Signals}}
  <table>
    <thead><tr><th>Symbol</th><th>Action</th><th>Fast</th><th>Slow</th><th>Reason</th></tr></thead>
    <tbody>
    {{range .Run.Signals}}
      <tr><td>{{.Symbol}}</td><td>{{.Action}}</td><td>{{f4 .Fast}}</td><td>{{f4 .Slow}}</td><td>{{redact .Reason}}</td></tr>
    {{end}}
    </tbody>
  </table>
  {{else}}<p>No signals.</p>{{end}}
  <h2>Paper fills</h2>
  {{if .Run.Fills}}
  <table>
    <thead><tr><th>Symbol</th><th>Side</th><th>Qty</th><th>Price</th><th>Fee</th></tr></thead>
    <tbody>
    {{range .Run.Fills}}
      <tr><td>{{.Symbol}}</td><td>{{.Side}}</td><td>{{f4 .Qty}}</td><td>{{f4 .Price}}</td><td>{{f4 .Fee}}</td></tr>
    {{end}}
    </tbody>
  </table>
  {{else}}<p>No paper fills in this run.</p>{{end}}
  <h2>Positions</h2>
  {{if .Run.Positions}}
  <table>
    <thead><tr><th>Symbol</th><th>Qty</th><th>Average</th></tr></thead>
    <tbody>
    {{range .Run.Positions}}
      <tr><td>{{.Symbol}}</td><td>{{f4 .Qty}}</td><td>{{f4 .AvgPrice}}</td></tr>
    {{end}}
    </tbody>
  </table>
  {{else}}<p>No open paper positions.</p>{{end}}
</body>
</html>
{{end}}
`
