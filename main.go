package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"embed"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

//go:embed frontend/dist
var assets embed.FS

const (
	CF_DIB = 8
)

type ClipboardItem struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type App struct {
	ctx              context.Context
	clipboardHistory []ClipboardItem
	lastContent      string
	lastImageHash    string
}

func NewApp() *App {
	return &App{
		clipboardHistory: make([]ClipboardItem, 0),
	}
}

func (a *App) initSysTray() {
	systray.Run(a.onSysTrayReady, a.onSysTrayExit)
}

func (a *App) onSysTrayExit() {
	// 清理工作
}

func (a *App) onSysTrayReady() {
	iconData, err := assets.ReadFile("build/icon.ico")
	if err != nil {
		fmt.Println("Failed to load icon:", err)
		return
	}
	systray.SetIcon(iconData) // 使用前面 //go:embed build/icon.ico 引入的图标
	systray.SetTitle("剪贴板历史")
	systray.SetTooltip("剪贴板历史工具")

	mShow := systray.AddMenuItem("显示主窗口", "显示主窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出应用")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
			}
		}
	}()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.watchClipboard()
	go a.initSysTray() // 初始化系统托盘
}

func (a *App) createMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// 文件菜单
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("显示主窗口", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
		runtime.WindowShow(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowSetAlwaysOnTop(a.ctx, false)
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("退出", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})

	return appMenu
}

// 处理窗口事件
func (a *App) handleWindowEvents() {
	// 监听窗口关闭事件
	runtime.EventsOn(a.ctx, "window-close-requested", func(data ...interface{}) {
		// 最小化到托盘而不是真正关闭
		runtime.Hide(a.ctx)
	})
}

// WindowHide 隐藏窗口
func (a *App) WindowHide() {
	runtime.Hide(a.ctx)
}

// WindowShow 显示窗口
func (a *App) WindowShow() {
	runtime.Show(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// 设置自启动
func (a *App) SetAutoStart(enable bool) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if err != nil {
		// 如果打开失败，尝试创建
		key, _, err = registry.CreateKey(
			registry.CURRENT_USER,
			`SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
			registry.QUERY_VALUE|registry.SET_VALUE,
		)
		if err != nil {
			return fmt.Errorf("无法访问注册表: %v", err)
		}
	}
	defer key.Close()

	// 获取可执行文件路径
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取程序路径: %v", err)
	}
	exePath, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("无法获取程序完整路径: %v", err)
	}

	appName := "剪贴板历史"

	if enable {
		// 添加到自启动
		err = key.SetStringValue(appName, exePath)
		if err != nil {
			return fmt.Errorf("设置自启动失败: %v", err)
		}
	} else {
		// 从自启动中移除
		err = key.DeleteValue(appName)
		if err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("取消自启动失败: %v", err)
		}
	}
	return nil
}

// 检查是否已设置自启动
func (a *App) CheckAutoStart() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	appName := "剪贴板历史"
	value, _, err := key.GetStringValue(appName)
	if err != nil {
		return false
	}

	// 检查路径是否匹配当前程序
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exePath, err := filepath.Abs(exe)
	if err != nil {
		return false
	}

	return strings.EqualFold(value, exePath)
}

func (a *App) getClipboardImage() (string, string, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	//gdi32 := syscall.NewLazyDLL("gdi32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	openClipboard := user32.NewProc("OpenClipboard")
	closeClipboard := user32.NewProc("CloseClipboard")
	getClipboardData := user32.NewProc("GetClipboardData")
	isClipboardFormatAvailable := user32.NewProc("IsClipboardFormatAvailable")

	// 检查是否有位图格式的数据
	ret, _, _ := isClipboardFormatAvailable.Call(uintptr(CF_DIB))
	if ret == 0 {
		return "", "", fmt.Errorf("no image in clipboard")
	}

	// 打开剪贴板
	ret, _, _ = openClipboard.Call(0)
	if ret == 0 {
		return "", "", fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard.Call()

	// 获取剪贴板数据
	handle, _, _ := getClipboardData.Call(uintptr(CF_DIB))
	if handle == 0 {
		return "", "", fmt.Errorf("failed to get clipboard data")
	}

	// 获取全局内存
	globalLock := kernel32.NewProc("GlobalLock")
	globalUnlock := kernel32.NewProc("GlobalUnlock")

	dataPtr, _, _ := globalLock.Call(handle)
	if dataPtr == 0 {
		return "", "", fmt.Errorf("failed to lock memory")
	}
	defer globalUnlock.Call(handle)

	// 读取 BITMAPINFOHEADER
	bmi := (*BITMAPINFOHEADER)(unsafe.Pointer(dataPtr))

	// 创建图像
	img := image.NewRGBA(image.Rect(0, 0, int(bmi.Width), int(bmi.Height)))

	// 复制像素数据
	for y := 0; y < int(bmi.Height); y++ {
		for x := 0; x < int(bmi.Width); x++ {
			offset := y*int(bmi.Width)*4 + x*4
			ptr := dataPtr + uintptr(unsafe.Sizeof(*bmi)) + uintptr(offset)
			b := *(*uint8)(unsafe.Pointer(ptr))
			g := *(*uint8)(unsafe.Pointer(ptr + 1))
			r := *(*uint8)(unsafe.Pointer(ptr + 2))
			a := *(*uint8)(unsafe.Pointer(ptr + 3))
			img.Set(x, int(bmi.Height)-y-1, color.RGBA{r, g, b, a})
		}
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", err
	}

	// 转换为 base64
	base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	hash := fmt.Sprintf("%x", md5.Sum(buf.Bytes()))

	return base64Data, hash, nil
}

// 新增辅助方法
func (a *App) addHistoryItem(itemType, content, hash string) {
	item := ClipboardItem{
		Type:      itemType,
		Content:   content,
		Timestamp: time.Now(),
	}

	if itemType == "text" {
		a.lastContent = content
	} else {
		a.lastImageHash = hash
	}

	a.clipboardHistory = append([]ClipboardItem{item}, a.clipboardHistory...)
	if len(a.clipboardHistory) > 100 {
		a.clipboardHistory = a.clipboardHistory[:100]
	}
	runtime.EventsEmit(a.ctx, "clipboard-update", a.clipboardHistory)
}
func (a *App) watchClipboard() {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// 文本检查
		text, err := runtime.ClipboardGetText(a.ctx)
		if err == nil && text != "" && text != a.lastContent {
			// 有新的文本内容
			a.addHistoryItem("text", text, "")
			continue
		}

		// 图片检查但不清空原有内容
		imgData, hash, err := a.getClipboardImage()
		if err == nil && imgData != "" && hash != a.lastImageHash {
			// 有新的图片内容
			a.addHistoryItem("image", imgData, hash)
		}
	}
}

// BITMAPINFOHEADER 结构体
type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func (a *App) GetClipboardHistory() []ClipboardItem {
	return a.clipboardHistory
}

func (a *App) SetClipboardContent(content string) error {
	return runtime.ClipboardSetText(a.ctx, content)
}

func (a *App) ClearHistory() {
	a.clipboardHistory = make([]ClipboardItem, 0)
	runtime.EventsEmit(a.ctx, "clipboard-update", a.clipboardHistory)
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "剪贴板历史",
		Width:     900,
		Height:    600,
		Assets:    assets,
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Menu: app.createMenu(),
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		OnDomReady: func(ctx context.Context) {
			app.handleWindowEvents()
		},
		HideWindowOnClose: true,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
