package model

// Device 表示一台被 adb 识别到的设备。
type Device struct {
	Serial         string `json:"serial"`
	Model          string `json:"model"`
	Brand          string `json:"brand"`
	AndroidVersion string `json:"androidVersion"`
	State          string `json:"state"`     // device | offline | unauthorized
	Transport      string `json:"transport"` // usb | wifi
	LastSeen       int64  `json:"lastSeen"`
}

// AdbEnv 是 EnsureAdbReady 返回的环境信息。
type AdbEnv struct {
	BinaryPath string `json:"binaryPath"`
	Version    string `json:"version"`
	ServerPort int    `json:"serverPort"`
	Ready      bool   `json:"ready"`
}
