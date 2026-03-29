package main

import (
	"embed"
	_ "embed"
	"iOSGhostRun/services"
	"log"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
	application.RegisterEvent[string]("developer-mode-menu-revealed")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.

	// 创建服务实例
	loggerSvc := services.NewLoggerService()
	devicesSvc := services.NewDevicesService()
	locationSvc := services.NewLocationService()
	runningSvc := services.NewRunningService(locationSvc)

	app := application.New(application.Options{
		Name:        "iOSGhostRun",
		Description: "iOS虚拟定位跑步应用",
		LogLevel:    slog.LevelInfo,
		Services: []application.Service{
			application.NewService(loggerSvc),
			application.NewService(devicesSvc),
			application.NewService(locationSvc),
			application.NewService(runningSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		OnShutdown: func() {
			devInfo, err := devicesSvc.GetSelectedDevice()
			if err != nil {
				services.Log.Debug("Main", "没有选中的设备需要清理")
				return
			}
			services.Log.Debug("Main", "正在清理选中的设备: "+devInfo.UDID)

			if err := locationSvc.ResetLocation(devInfo.UDID); err != nil {
				services.Log.Error("Main", "重置设备位置失败: "+err.Error())
			}

			if err := services.UnmountImage(devInfo.UDID); err != nil {
				services.Log.Error("Main", "卸载开发者镜像失败: "+err.Error())
			}

		},
	})

	app.SetIcon(icon)
	var allowQuit atomic.Bool

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
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
	})

	// Run the application. This blocks until the application has been exited.
	err := app.Run()
	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
