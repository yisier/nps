export namespace main {
	
	export class ConnectionLog {
	    timestamp: string;
	    message: string;
	    type: string;
	    clientId: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.message = source["message"];
	        this.type = source["type"];
	        this.clientId = source["clientId"];
	    }
	}
	export class GuiSettings {
	    startupEnabled: boolean;
	    rememberClientState: boolean;
	    logDir: string;
	    themeMode: string;
	
	    static createFrom(source: any = {}) {
	        return new GuiSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startupEnabled = source["startupEnabled"];
	        this.rememberClientState = source["rememberClientState"];
	        this.logDir = source["logDir"];
	        this.themeMode = source["themeMode"];
	    }
	}
	export class ShortClient {
	    name: string;
	    addr: string;
	    key: string;
	    tls: boolean;
	    running: boolean;
	    error: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ShortClient(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.addr = source["addr"];
	        this.key = source["key"];
	        this.tls = source["tls"];
	        this.running = source["running"];
	        this.error = source["error"];
	        this.status = source["status"];
	    }
	}

}

