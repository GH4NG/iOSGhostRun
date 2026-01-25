package main

import (
	"embed"
	_ "embed"
	"iOSGhostRun/services"
	"log"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
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
}

// package-level service references used by cleanup
var (
	devicesSvc  *services.DevicesService
	locationSvc *services.LocationService
	imageSvc    *services.ImageService
)

func cleanup() {
	services.Log.Debug("Main", "正在清理：重置位置和卸载镜像")
	devInfo, err := devicesSvc.GetSelectedDevice()
	if err != nil {
		services.Log.Debug("Main", "没有选中的设备需要清理")
		return
	}

	udid := devInfo.UDID

	// 执行重置和卸载操作
	go func() {
		if err := locationSvc.ResetLocation(udid); err != nil {
			services.Log.Error("Main", "重置位置失败: "+err.Error())
		}
	}()

	go func() {
		if err := imageSvc.UnmountImage(udid); err != nil {
			services.Log.Error("Main", "卸载开发镜像失败: "+err.Error())
		}
	}()
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
	devicesSvc = &services.DevicesService{}
	locationSvc = &services.LocationService{}
	runningSvc := services.NewRunningService()
	imageSvc = &services.ImageService{}

	app := application.New(application.Options{
		Name:        "iOSGhostRun",
		Description: "iOS虚拟定位跑步应用",
		LogLevel:    slog.LevelInfo,
		Services: []application.Service{
			application.NewService(loggerSvc),
			application.NewService(devicesSvc),
			application.NewService(locationSvc),
			application.NewService(runningSvc),
			application.NewService(imageSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		OnShutdown: func() {
			cleanup()
		},
	})

	app.SetIcon(icon)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "Main", Title: "iOS虚拟定位跑步",
		Width: 800, Height: 600,

		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},

		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// 将窗口附加到桌面层
	// desktopWindowTool := NewDesktopWindowTool(window)
	// desktopWindowTool.AttachToDesktop()

	sysTray := app.SystemTray.New()
	sysTray.SetIcon(icon)
	sysTray.SetLabel("iOSGhostRun")
	sysTray.SetTooltip("iOS虚拟定位跑步")
	sysTray.OnClick(func() {
		if window != nil {
			window.Show()
			window.Focus()
		}
	})
	trayMenu := app.Menu.New()
	sysTray.SetMenu(trayMenu)
	trayMenu.Add("打开").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	// Run the application. This blocks until the application has been exited.
	err := app.Run()
	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
