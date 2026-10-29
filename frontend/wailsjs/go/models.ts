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

