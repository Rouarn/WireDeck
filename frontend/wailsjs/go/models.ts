export namespace model {
	
	export class AdbEnv {
	    binaryPath: string;
	    version: string;
	    serverPort: number;
	    ready: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AdbEnv(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.binaryPath = source["binaryPath"];
	        this.version = source["version"];
	        this.serverPort = source["serverPort"];
	        this.ready = source["ready"];
	    }
	}
	export class ApkMeta {
	    path: string;
	    label: string;
	    packageName: string;
	    versionName: string;
	    versionCode: number;
	    minSdk: number;
	    targetSdk: number;
	    abis: string[];
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new ApkMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.label = source["label"];
	        this.packageName = source["packageName"];
	        this.versionName = source["versionName"];
	        this.versionCode = source["versionCode"];
	        this.minSdk = source["minSdk"];
	        this.targetSdk = source["targetSdk"];
	        this.abis = source["abis"];
	        this.size = source["size"];
	    }
	}
	export class AppDetail {
	    package: string;
	    versionName: string;
	    versionCode: string;
	    minSdk: string;
	    targetSdk: string;
	    firstInstall: string;
	    lastUpdate: string;
	    dataDir: string;
	    permissions: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.package = source["package"];
	        this.versionName = source["versionName"];
	        this.versionCode = source["versionCode"];
	        this.minSdk = source["minSdk"];
	        this.targetSdk = source["targetSdk"];
	        this.firstInstall = source["firstInstall"];
	        this.lastUpdate = source["lastUpdate"];
	        this.dataDir = source["dataDir"];
	        this.permissions = source["permissions"];
	    }
	}
	export class AppOpResult {
	    ok: boolean;
	    message: string;
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new AppOpResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.raw = source["raw"];
	    }
	}
	export class AppSummary {
	    package: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.package = source["package"];
	    }
	}
	export class ConnectRequest {
	    host: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	    }
	}
	export class ConnectResult {
	    ok: boolean;
	    address: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.address = source["address"];
	        this.message = source["message"];
	    }
	}
	export class Device {
	    serial: string;
	    model: string;
	    brand: string;
	    androidVersion: string;
	    state: string;
	    transport: string;
	    hwSerial: string;
	    lastSeen: number;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial = source["serial"];
	        this.model = source["model"];
	        this.brand = source["brand"];
	        this.androidVersion = source["androidVersion"];
	        this.state = source["state"];
	        this.transport = source["transport"];
	        this.hwSerial = source["hwSerial"];
	        this.lastSeen = source["lastSeen"];
	    }
	}
	export class HostIdentity {
	    hostname: string;
	    lanIP: string;
	
	    static createFrom(source: any = {}) {
	        return new HostIdentity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.lanIP = source["lanIP"];
	    }
	}
	export class InstallOptions {
	    replace: boolean;
	    allowDowngrade: boolean;
	    grantPermissions: boolean;
	    allowTest: boolean;
	    userId?: number;
	
	    static createFrom(source: any = {}) {
	        return new InstallOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.replace = source["replace"];
	        this.allowDowngrade = source["allowDowngrade"];
	        this.grantPermissions = source["grantPermissions"];
	        this.allowTest = source["allowTest"];
	        this.userId = source["userId"];
	    }
	}
	export class InstallRequest {
	    serial: string;
	    paths: string[];
	    options: InstallOptions;
	
	    static createFrom(source: any = {}) {
	        return new InstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial = source["serial"];
	        this.paths = source["paths"];
	        this.options = this.convertValues(source["options"], InstallOptions);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MDNSService {
	    name: string;
	    kind: string;
	    address: string;
	    host: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new MDNSService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.address = source["address"];
	        this.host = source["host"];
	        this.port = source["port"];
	    }
	}
	export class PairRequest {
	    host: string;
	    pairPort: number;
	    code: string;
	
	    static createFrom(source: any = {}) {
	        return new PairRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.pairPort = source["pairPort"];
	        this.code = source["code"];
	    }
	}
	export class PairResult {
	    ok: boolean;
	    message: string;
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new PairResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.raw = source["raw"];
	    }
	}

}

