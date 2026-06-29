// Package manager hosts only the Wails binding shim. The real conversion
// orchestration lives in open-make-tiff/internal/app; this type delegates to it
// and adapts the framework-agnostic Progress interface to the Wails events the
// frontend expects (omt:convert:*).
package manager

import (
	"context"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"

	"open-make-tiff/internal/app"
	"open-make-tiff/internal/config"
)

// Api is the object bound to the frontend via Wails. It must stay named Api and
// stay in package manager so the generated frontend binding path
// (wailsjs/go/manager/Api.js) does not change.
type Api struct {
	omt *app.App
	ctx context.Context // injected from OnStartup
}

// NewApi wires the binding shim to the framework-agnostic App.
func NewApi(omt *app.App) *Api { return &Api{omt: omt} }

// SetContext injects the Wails runtime context (called from OnStartup).
func (a *Api) SetContext(ctx context.Context) { a.ctx = ctx }

// Context exposes the Wails runtime context (used by OnSecondInstanceLaunch).
func (a *Api) Context() context.Context { return a.ctx }

// GetSetting returns the selectable option lists for the frontend dropdowns.
func (a *Api) GetSetting() *config.Setting { return config.NewSetting() }

// GetConfig returns the current persisted configuration.
func (a *Api) GetConfig() *config.Config { return a.omt.GetConfig() }

// SetConfig persists the config and re-applies the always-on-top window state.
func (a *Api) SetConfig(cfg *config.Config) *config.Config {
	res := a.omt.SetConfig(cfg)
	wails_runtime.WindowSetAlwaysOnTop(a.ctx, res.EnableWindowTop)
	return res
}

// Convert runs a batch conversion, translating Progress callbacks into the
// omt:convert:* events the frontend listens for.
func (a *Api) Convert(paths []string) {
	a.omt.Convert(a.ctx, paths, &wailsProgress{ctx: a.ctx})
}

// wailsProgress adapts app.Progress to Wails events consumed by App.vue.
type wailsProgress struct{ ctx context.Context }

func (p *wailsProgress) Start(int) { wails_runtime.EventsEmit(p.ctx, "omt:convert:started") }

func (p *wailsProgress) File(path string, r app.Result, _ error) {
	switch r {
	case app.ResultProcessing:
		wails_runtime.EventsEmit(p.ctx, "omt:convert:file:started", path)
	case app.ResultSucceeded:
		wails_runtime.EventsEmit(p.ctx, "omt:convert:file:success", path)
	case app.ResultSkipped:
		wails_runtime.EventsEmit(p.ctx, "omt:convert:file:skipped", path)
	case app.ResultFailed:
		wails_runtime.EventsEmit(p.ctx, "omt:convert:file:error", path)
	}
}

func (p *wailsProgress) Done() { wails_runtime.EventsEmit(p.ctx, "omt:convert:finished") }
