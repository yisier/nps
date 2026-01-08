export namespace main {
	
	export class ShortClient {
	    name: string;
	    addr: string;
	    key: string;
	    tls: boolean;
	    running: boolean;
	
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
	    }
	}

}

