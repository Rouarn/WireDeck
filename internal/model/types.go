package model

// Device 表示一台被 adb 识别到的设备。
type Device struct {
	Serial         string `json:"serial"`
	Model          string `json:"model"`
	Brand          string `json:"brand"`
	AndroidVersion string `json:"androidVersion"`
	State          string `json:"state"`     // device | offline | unauthorized
	Transport      string `json:"transport"` // usb | wifi
	HwSerial       string `json:"hwSerial"`  // ro.serialno，用于合并同一设备的多种连接
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

// HostIdentity 是电脑在手机端"已配对设备"列表中的身份信息。
type HostIdentity struct {
	Hostname string `json:"hostname"` // 配对时上报的主机名
	LANIP    string `json:"lanIP"`    // 电脑局域网 IP
}

// ApkMeta 是安装前解析出的 APK 信息。
type ApkMeta struct {
	Path        string   `json:"path"`
	Label       string   `json:"label"`
	PackageName string   `json:"packageName"`
	VersionName string   `json:"versionName"`
	VersionCode int      `json:"versionCode"`
	MinSDK      int      `json:"minSdk"`
	TargetSDK   int      `json:"targetSdk"`
	ABIs        []string `json:"abis"`
	Size        int64    `json:"size"`
}

// InstallOptions 是安装参数（对应 adb install 的 flags）。
type InstallOptions struct {
	Replace          bool `json:"replace"`          // -r
	AllowDowngrade   bool `json:"allowDowngrade"`   // -d
	GrantPermissions bool `json:"grantPermissions"` // -g
	AllowTest        bool `json:"allowTest"`        // -t
	UserID           *int `json:"userId"`           // --user
}

// InstallRequest 是一次安装请求。
type InstallRequest struct {
	Serial  string         `json:"serial"`
	Paths   []string       `json:"paths"`
	Options InstallOptions `json:"options"`
}

// TaskProgress 是安装/传输任务的阶段进度。
type TaskProgress struct {
	TaskID  string `json:"taskId"`
	Stage   string `json:"stage"`   // parse | transfer | install | verify | done | error
	Percent int    `json:"percent"` // -1 表示不可量化
	Message string `json:"message"`
}

// InstallResult 是一次安装的最终结果。
type InstallResult struct {
	TaskID     string `json:"taskId"`
	OK         bool   `json:"ok"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Advice     string `json:"advice"`
	Raw        string `json:"raw"`
	DurationMs int64  `json:"durationMs"`
}

// AppSummary 是应用列表中的一项。
type AppSummary struct {
	Package string `json:"package"`
}

// AppDetail 是应用详情。
type AppDetail struct {
	Package      string   `json:"package"`
	VersionName  string   `json:"versionName"`
	VersionCode  string   `json:"versionCode"`
	MinSDK       string   `json:"minSdk"`
	TargetSDK    string   `json:"targetSdk"`
	FirstInstall string   `json:"firstInstall"`
	LastUpdate   string   `json:"lastUpdate"`
	DataDir      string   `json:"dataDir"`
	Permissions  []string `json:"permissions"`
}

// AppOpResult 是应用操作（卸载 / 启动 / 强停 / 清数据 / 导出）的结果。
type AppOpResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
}
