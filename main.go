package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"iOSGhostRun/internal/device"
	"iOSGhostRun/internal/location"
	"iOSGhostRun/internal/logger"
	"iOSGhostRun/internal/runner"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 创建服务实例
	logService := logger.NewService()
	deviceManager := device.NewManager(logService)
	locationService := location.NewService()
	runnerService := runner.NewService(deviceManager, locationService, logService)

	// 创建Wails应用
	err := wails.Run(&options.App{
		Title:     "iOSGhostRun - iOS虚拟跑步",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			logService.Startup(ctx)
			deviceManager.Startup(ctx)
			runnerService.Startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			deviceManager.StopTunnel()
			runnerService.Shutdown(ctx)
		},
		Bind: []interface{}{
			logService,
			deviceManager,
			locationService,
			runnerService,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
