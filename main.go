package main

import (
	"embed"
	_ "embed"
	"iOSGhostRun/services"
	"log"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func init() {
	// 注册事件数据类型，让 Wails 生成对应的 TypeScript 绑定。
	application.RegisterEvent[services.RunningStatus]("running:position")
	application.RegisterEvent[services.RunningStatus]("running:completed")
	application.RegisterEvent[string]("running:error")
	application.RegisterEvent[services.LogEntry]("log-event")
	application.RegisterEvent[string]("developer-mode-menu-revealed")
	application.RegisterEvent[application.Void]("app:close-requested")
	application.RegisterEvent[application.Void]("app:close-quit")
}

func main() {
	// 创建服务实例
	loggerSvc := services.NewLoggerService()
	devicesSvc := services.NewDevicesService()
	locationSvc := services.NewLocationService()
	runningSvc := services.NewRunningService(locationSvc)
	updateSvc := services.NewUpdateService()

	var window *application.WebviewWindow
	var allowQuit atomic.Bool
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			services.SetAppShuttingDown(true)
			runningSvc.StopRun()
			devInfo, err := devicesSvc.GetSelectedDevice()
			if err == nil {
				_ = services.UnmountImage(devInfo.UDID)
			}
			_ = services.StopTunnel()
		})
	}

	app := application.New(application.Options{
		Name:        "iOSGhostRun",
		Description: "iOS虚拟定位跑步应用",
		LogLevel:    slog.LevelInfo,
		ShouldQuit: func() bool {
			cleanup()
			return true
		},
		// 二次启动时唤起已有窗口
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.iosghostrun.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Focus()
				}
			},
		},
		Services: []application.Service{
			application.NewService(loggerSvc),
			application.NewService(devicesSvc),
			application.NewService(locationSvc),
			application.NewService(runningSvc),
			application.NewService(updateSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			ApplicationID: "com.iosghostrun.app",
		},
	})
	if err := services.ConfigureUpdateService(updateSvc, app.Updater); err != nil {
		log.Fatalf("配置应用更新失败: %v", err)
	}

	app.SetIcon(icon)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "iOS虚拟定位跑步",
		Width: 800, Height: 600,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(27, 38, 54, 230),

		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
			TitleBar: application.MacTitleBarHidden,
		},

		URL: "/",
	})

	window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if allowQuit.Load() {
			return
		}
		event.Cancel()
		app.Event.Emit("app:close-requested")
	})

	app.Event.On("app:close-quit", func(_ *application.CustomEvent) {
		if allowQuit.Load() {
			return
		}
		allowQuit.Store(true)
		cleanup()
		window.Close()
	})

	// Run the application. This blocks until the application has been exited.
	err := app.Run()
	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
