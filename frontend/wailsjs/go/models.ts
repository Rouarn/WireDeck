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
	export class Device {
	    serial: string;
	    model: string;
	    brand: string;
	    androidVersion: string;
	    state: string;
	    transport: string;
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
	        this.lastSeen = source["lastSeen"];
	    }
	}

}

