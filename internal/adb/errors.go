package adb

import (
	"errors"
	"strings"
)

// 错误集合。上层（Wails 绑定）会把这些错误转成前端可见的提示。
var (
	ErrAdbNotFound = errors.New("未找到 adb 可执行文件，请安装 Android platform-tools，或在设置中手动指定路径")
	ErrAdbServer   = errors.New("adb server 启动失败")
	ErrTimeout     = errors.New("adb 命令执行超时")
	ErrNoDevice    = errors.New("没有可用的设备")
	ErrCanceled    = errors.New("操作已取消")
	ErrPushFailed  = errors.New("传输文件到设备失败")
)

// installAdvice 是安装失败码到中文处理建议的映射（对应文档第 8 章）。
var installAdvice = map[string]string{
	"INSTALL_FAILED_ALREADY_EXISTS":             "已存在同一包名，勾选「覆盖安装（-r）」后重试",
	"INSTALL_FAILED_VERSION_DOWNGRADE":          "安装包版本低于设备已装版本，勾选「允许降级（-d）」或先卸载旧版",
	"INSTALL_FAILED_UPDATE_INCOMPATIBLE":        "与已安装版本签名不一致，需先卸载旧版再安装",
	"INSTALL_FAILED_INSUFFICIENT_STORAGE":       "设备存储空间不足，请清理后重试",
	"INSTALL_FAILED_INVALID_APK":                "安装包损坏或不完整，请重新获取文件",
	"INSTALL_FAILED_NO_MATCHING_ABIS":           "安装包不包含设备所需的 CPU 架构，请换用对应 ABI 的包",
	"INSTALL_FAILED_CONFLICTING_PROVIDER":       "与设备上其他应用存在 Provider 冲突，请卸载冲突应用",
	"INSTALL_FAILED_DUPLICATE_PERMISSION":       "权限声明重复，请卸载旧版或修改 APK",
	"INSTALL_FAILED_SHARED_USER_INCOMPATIBLE":   "sharedUserId 签名不匹配，需系统签名或卸载重装",
	"INSTALL_FAILED_USER_RESTRICTED":            "设备限制了安装：小米/Redmi 需开启「USB 安装」，OPPO/vivo 类似开关",
	"INSTALL_FAILED_ABORTED":                    "安装被中止，请确认设备未锁屏、未拒绝安装弹窗",
	"INSTALL_FAILED_TEST_ONLY":                  "这是测试包，勾选「允许测试包（-t）」后重试",
	"INSTALL_PARSE_FAILED_NO_CERTIFICATES":      "安装包未签名，请使用已签名版本",
	"INSTALL_PARSE_FAILED_MANIFEST_MALFORMED":   "Manifest 格式非法，安装包可能被二次修改损坏",
	"INSTALL_FAILED_DEPRECATED_SDK_VERSION":     "目标 SDK 过低，当前系统版本无法安装",
	"INSTALL_FAILED_MEDIA_UNAVAILABLE":          "设备存储不可用，请检查是否已挂载或解密",
	"INSTALL_FAILED_UID_CHANGED":                "应用 UID 发生变化，卸载后重新安装",
	"INSTALL_FAILED_VERIFICATION_FAILURE":       "系统安装校验失败，可尝试关闭「验证应用」后重试",
	"INSTALL_FAILED_CONTAINER_ERROR":            "安装容器错误，重启设备后重试",
	"INSTALL_FAILED_INTERNAL_ERROR":             "系统内部错误，重启设备后重试",
	"INSTALL_FAILED_MISSING_SHARED_LIBRARY":     "缺少共享库，设备系统版本可能不兼容",
	"INSTALL_FAILED_DUPLICATE_PACKAGE":          "已存在同名包，先卸载或使用覆盖安装",
	"INSTALL_FAILED_DEVICE_POLICY_MANAGER":      "设备策略禁止安装，需管理员权限",
	"INSTALL_FAILED_INVALID_INSTALL_LOCATION":   "安装位置无效，请检查存储设置",
	"INSTALL_FAILED_PERMISSION_MODEL_DOWNGRADE": "权限模型降级不被允许，请卸载重装",
}

// ExplainInstallFailure 从 adb install 输出中解析失败码，并给出处理建议。
//
// 输出示例：
//
//	Failure [INSTALL_FAILED_UPDATE_INCOMPATIBLE: Package com.example signatures do not match]
func ExplainInstallFailure(output string) (code, advice string) {
	const marker = "Failure ["

	idx := strings.Index(output, marker)
	if idx < 0 {
		return "", ""
	}

	rest := output[idx+len(marker):]
	end := strings.IndexAny(rest, ":]")
	if end < 0 {
		end = len(rest)
	}

	code = strings.TrimSpace(rest[:end])
	advice = installAdvice[code]
	if advice == "" {
		advice = "未识别的错误码，请查看原始输出，或重启设备后重试"
	}
	return code, advice
}
