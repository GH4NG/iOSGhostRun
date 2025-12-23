# iOSGhostRun - iOS 虚拟跑步

基于 go-ios 和 Wails 开发的跨平台 iOS 虚拟定位跑步打卡应用。

## 功能特性

-   **虚拟跑步模拟**: 模拟真实跑步轨迹，支持设置跑步速度
-   **随机化支持**: 支持速度随机波动和路线随机偏移，模拟真实跑步
-   **路线规划**: 支持在地图上手动绘制路线，自动保存上次路线
-   **坐标系转换**: 支持高德/百度/GPS 坐标系自动转换
-   **设备管理**: 自动检测并连接 iOS 设备
-   **跨平台**: 支持 Windows、macOS、Linux

## 系统要求

-   Go 1.21+
-   Node.js 18+
-   Wails CLI v2
-   iOS 设备需通过 USB 连接并信任此电脑
-   iOS 17+ 设备需要先运行 `ios tunnel start`（需管理员权限）

## 安装依赖

### 1. 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 2. 安装前端依赖

```bash
cd frontend
npm install
```

### 3. 安装 Go 依赖

```bash
go mod tidy
```

## 开发运行

```bash
wails dev
```

## 构建发布

### Windows

```bash
wails build -platform windows/amd64
```

### macOS

```bash
wails build -platform darwin/universal
```

### Linux

```bash
wails build -platform linux/amd64
```

## 使用说明

### 1. 连接设备

1. 使用 USB 连接 iOS 设备到电脑
2. 在设备上点击"信任此电脑"
3. 点击应用中的"刷新"按钮
4. 选择要使用的设备

### 2. 绘制路线

1. 点击地图右上角的 D 按钮进入绘制模式
2. 在地图上点击添加路线点（至少 2 个点）
3. 路线会自动保存，下次启动时自动加载

### 3. 配置跑步参数

-   **跑步速度**: 3-20 km/h
-   **速度波动**: 模拟真实跑步的速度变化
-   **路线偏移**: 模拟 GPS 定位抖动
-   **循环次数**: 设置跑几圈，0 表示无限循环

### 4. 开始跑步

1. 确保设备已连接、路线已设置
2. 点击"开始跑步"
3. 可随时暂停/继续/停止
4. 跑步完成后点击"重置真实位置"恢复设备实际位置

## iOS 17+ 特别说明

iOS 17 及以上版本需要先启动隧道服务：

**Windows (管理员权限运行):**

1. 下载 wintun.dll: https://www.wintun.net/
2. 将 wintun.dll 复制到 C:\Windows\System32
3. 运行 `ios tunnel start`

**macOS/Linux (sudo):**

```bash
sudo ios tunnel start
```

## 注意事项

**免责声明**: 本软件仅供学习和研究使用。使用虚拟定位可能违反某些应用的服务条款，请自行承担使用风险。

## 技术栈

-   **后端**: Go + go-ios
-   **前端**: Vue 3 + Leaflet
-   **桌面框架**: Wails v2

## 许可证

MIT License
