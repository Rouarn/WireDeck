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

// MDNSService 是 `adb mdns services` 发现的一条服务记录。
type MDNSService struct {
	Name    string `json:"name"`    // adb-xxxx
	Kind    string `json:"kind"`    // pairing | connect | other
	Address string `json:"address"` // host:port
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

// PairRequest 是无线调试配对请求。
type PairRequest struct {
	Host     string `json:"host"`
	PairPort int    `json:"pairPort"`
	Code     string `json:"code"`
}

// PairResult 是配对结果。
type PairResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
}

// ConnectRequest 是连接请求（连接端口与配对端口不同）。
type ConnectRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ConnectResult 是连接结果。
type ConnectResult struct {
	OK      bool   `json:"ok"`
	Address string `json:"address"`
	Message string `json:"message"`
}
