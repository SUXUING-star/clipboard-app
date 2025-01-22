package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 监听剪贴板的方法
func (a *App) GetClipboardText() string {
	text, err := runtime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	return text
}

// 写入剪贴板的方法
func (a *App) SetClipboardText(text string) error {
	return runtime.ClipboardSetText(a.ctx, text)
}

// 获取剪贴板图片的方法
func (a *App) GetClipboardImage() (string, error) {
	// TODO: 实现图片获取逻辑
	return "", nil
}

// 写入图片到剪贴板的方法
func (a *App) SetClipboardImage(imageData string) error {
	// TODO: 实现图片写入逻辑
	return nil
}
