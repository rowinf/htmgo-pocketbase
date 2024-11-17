package main

import (
	"fmt"
	"htmgopocketbase/__htmgo"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maddalax/htmgo/framework/config"
	h "github.com/maddalax/htmgo/framework/h"
	"github.com/maddalax/htmgo/framework/service"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type AppOpts struct {
	LiveReload     bool
	ServiceLocator *service.Locator
	Register       func(app *App)
}

type App struct {
	Opts   AppOpts
	Router *chi.Mux
}

const RequestContextKey = "htmgo.request.context"

func (app *App) start() {
	slog.SetLogLoggerLevel(h.GetLogLevel())

	pb := pocketbase.New()
	pb.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
		return nil
	})
	// app.Router.Use(func(h http.Handler) http.Handler {
	// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		cc := &hg.RequestContext{
	// 			locator:  app.Opts.ServiceLocator,
	// 			Request:  r,
	// 			Response: w,
	// 			kv:       make(map[string]interface{}),
	// 		}
	// 		populateHxFields(cc)
	// 		ctx := context.WithValue(r.Context(), RequestContextKey, cc)
	// 		h.ServeHTTP(w, r.WithContext(ctx))
	// 	})
	// })

	if app.Opts.Register != nil {
		app.Opts.Register(app)
	}

	// if app.Opts.LiveReload && h.IsDevelopment() {
	// 	app.AddLiveReloadHandler("/dev/livereload")
	// }

	port := ":3000"
	slog.Info(fmt.Sprintf("Server started at localhost%s", port))

	if err := pb.Start(); err != nil {
		slog.Error("failed to start pocketbase", err)
		panic(err)
	}
	if err := http.ListenAndServe(port, app.Router); err != nil {
		// If we are in watch mode, just try to kill any processes holding that port
		// and try again
		if h.IsDevelopment() && h.IsWatchMode() {
			slog.Info("Port already in use, trying to kill the process and start again")
			if runtime.GOOS == "windows" {
				cmd := exec.Command("cmd", "/C", fmt.Sprintf(`for /F "tokens=5" %%i in ('netstat -aon ^| findstr :%s') do taskkill /F /PID %%i`, port))
				cmd.Run()
			} else {
				cmd := exec.Command("bash", "-c", fmt.Sprintf("kill -9 $(lsof -ti%s)", port))
				cmd.Run()
			}

			time.Sleep(time.Millisecond * 50)

			// Try to start server again
			if err := http.ListenAndServe(port, app.Router); err != nil {
				slog.Error("Failed to restart server", "error", err)
				panic(err)
			}
		}

		panic(err)
	}
}

// Start starts the htmgo server
func start(opts AppOpts) {
	router := chi.NewRouter()
	instance := App{
		Opts:   opts,
		Router: router,
	}
	instance.start()
}

func main() {
	locator := service.NewLocator()
	cfg := config.Get()

	start(AppOpts{
		ServiceLocator: locator,
		LiveReload:     true,
		Register: func(app *App) {
			sub, err := fs.Sub(GetStaticAssets(), "assets/dist")
			if err != nil {
				panic(err)
			}

			http.FileServerFS(sub)

			// change this in htmgo.yml (public_asset_path)
			app.Router.Handle(fmt.Sprintf("%s/*", cfg.PublicAssetPath),
				http.StripPrefix(cfg.PublicAssetPath, http.FileServerFS(sub)))

			__htmgo.Register(app.Router)
		},
	})
}
