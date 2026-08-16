package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"os"
	"runtime"
	"slices"
	"sync/atomic"
	"time"

	appservice "github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/config"
	"github.com/headscaleclient/headscaleclient/internal/daemon"
	"github.com/headscaleclient/headscaleclient/internal/desktop"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	tailscaleadapter "github.com/headscaleclient/headscaleclient/internal/tailscale"
	wails "github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	appName    = "HeadscaleClient"
	appVersion = "0.1.0"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func init() {
	wails.RegisterEvent[appservice.SnapshotChangedEvent]("app:snapshot-changed")
	wails.RegisterEvent[appservice.LoginURLEvent]("app:login-url")
	wails.RegisterEvent[appservice.LoginFinishedEvent]("app:login-finished")
	wails.RegisterEvent[appservice.OperationFailedEvent]("app:operation-failed")
	wails.RegisterEvent[desktop.NavigateEvent](desktop.EventNavigate)
}

func main() {
	store, err := config.NewStore()
	if err != nil {
		log.Fatal(err)
	}

	desktopRuntime := desktop.NewRuntime()
	service, err := appservice.NewService(
		tailscaleadapter.NewAdapter(),
		store,
		desktopRuntime,
		appservice.WithAutostart(desktopRuntime),
		appservice.WithDaemonLifecycle(daemon.NewManager()),
		appservice.WithDiagnostics(appVersion, "3.0.0-beta.8", "tailscaled LocalAPI", runtime.GOOS+"/"+runtime.GOARCH),
	)
	if err != nil {
		log.Fatal(err)
	}
	backend := desktop.NewBackend(service)

	backgroundLaunch := slices.Contains(os.Args[1:], "--background")
	var mainWindow *wails.WebviewWindow
	windowReady := make(chan struct{})
	var quitting atomic.Bool

	showMainWindow := func() {
		<-windowReady
		mainWindow.Show()
		mainWindow.Restore()
		mainWindow.Focus()
	}

	app := wails.New(wails.Options{
		Name:        appName,
		Description: "Cross-platform client for Tailscale-compatible networks",
		Icon:        appIcon,
		Services: []wails.Service{
			wails.NewService(backend),
		},
		Assets: wails.AssetOptions{
			Handler:        wails.AssetFileServerFS(assets),
			DisableLogging: true,
		},
		MarshalError: marshalError,
		SingleInstance: &wails.SingleInstanceOptions{
			UniqueID: "io.headscaleclient.desktop",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(wails.SecondInstanceData) {
				showMainWindow()
			},
		},
		ShouldQuit: func() bool {
			quitting.Store(true)
			return true
		},
		ErrorHandler: func(err error) {
			log.Printf("Wails: %v", err)
		},
		Mac: wails.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: wails.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		Linux: wails.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})
	desktopRuntime.Attach(app)

	mainWindow = app.Window.NewWithOptions(wails.WebviewWindowOptions{
		Name:                       "main",
		Title:                      appName,
		Width:                      960,
		Height:                     680,
		MinWidth:                   720,
		MinHeight:                  520,
		InitialPosition:            wails.WindowCentered,
		Hidden:                     backgroundLaunch,
		BackgroundColour:           wails.NewRGB(247, 249, 248),
		URL:                        "/",
		DefaultContextMenuDisabled: true,
	})
	close(windowReady)

	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *wails.WindowEvent) {
		if quitting.Load() {
			return
		}
		if shouldCloseToTray(store) {
			event.Cancel()
			mainWindow.Hide()
			return
		}
		quitting.Store(true)
		go app.Quit()
	})

	trayController := desktop.NewTrayController(app, service, desktop.TrayControllerOptions{
		Icon:       appIcon,
		ShowWindow: showMainWindow,
		Navigate: func(target desktop.NavigationTarget) {
			desktopRuntime.Emit(desktop.EventNavigate, desktop.NavigateEvent{View: target})
		},
		Quit: func() {
			quitting.Store(true)
			app.Quit()
		},
	})
	unsubscribeTray := desktopRuntime.Subscribe(trayController.HandleEvent)
	defer unsubscribeTray()
	go trayController.Refresh()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func shouldCloseToTray(store *config.Store) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	settings, err := store.GetSettings(ctx)
	if err != nil {
		return true
	}
	return settings.CloseToTray
}

func marshalError(err error) []byte {
	problem := domain.ProblemFromError(err)
	data, marshalErr := json.Marshal(problem)
	if marshalErr != nil {
		return nil
	}
	return data
}
