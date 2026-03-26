package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// BaseDataPath 指向系统数据保存的根本目录
var BaseDataPath = filepath.Join("..", "data")

// InitDataDir 初始化根目录结构，挂载点创建
func InitDataDir() error {
	p, _ := filepath.Abs(BaseDataPath)
	return os.MkdirAll(p, 0755)
}

// GetVolumeRealDir 验证并映射请求中的映射目录名，确保目录名符合规范并且获取在 OS 中的决定绝对路径
func GetVolumeRealDir(folderName string) (string, error) {
	// 简单防御：防止用奇怪的字符跳跃目录
	if strings.Contains(folderName, "/") || strings.Contains(folderName, "\\") || strings.Contains(folderName, "..") {
		return "", errors.New("非法的储存卷名称记录")
	}
	p := filepath.Join(BaseDataPath, folderName)
	return filepath.Abs(p)
}

// GetFileRealDir 安全地对用户提供的基于指定储存卷的相对路径进行清洗计算
func GetFileRealDir(folderName string, relativePath string) (string, error) {
	volDir, err := GetVolumeRealDir(folderName)
	if err != nil {
		return "", err
	}

	cleanRel := filepath.Clean(relativePath)
	// 防止 / 开始变成绝对路径或者包含 ..
	if strings.HasPrefix(cleanRel, "..") || filepath.IsAbs(relativePath) || strings.HasPrefix(relativePath, "/") {
		// 为了更加兼容前导的 "/", filepath.Abs 可能会改变盘符逻辑，最好手动将其去掉
		cleanRel = strings.TrimPrefix(filepath.Clean("/"+relativePath), "/")
		if strings.HasPrefix(cleanRel, "..") {
			return "", errors.New("不合法的相对层级，请求被安全模块阻断")
		}
	}

	target := filepath.Join(volDir, cleanRel)

	// 安全兜底判定 - 绝对目标路径必须是以指定的卷根目录开头
	if !strings.HasPrefix(target, volDir) {
		return "", errors.New("非法的系统跨卷访问行为")
	}

	return target, nil
}
