export namespace manager {
	
	export class Config {
	    disable_adobe_dng_converter?: boolean;
	    enable_window_top?: boolean;
	    enable_subfolder?: boolean;
	    enable_compression?: boolean;
	    icc_profile?: string;
	    workers?: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.disable_adobe_dng_converter = source["disable_adobe_dng_converter"];
	        this.enable_window_top = source["enable_window_top"];
	        this.enable_subfolder = source["enable_subfolder"];
	        this.enable_compression = source["enable_compression"];
	        this.icc_profile = source["icc_profile"];
	        this.workers = source["workers"];
	    }
	}
	export class ProfileOption {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class WorkerNumOption {
	    value: number;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkerNumOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class Setting {
	    worker_nums: WorkerNumOption[];
	    profiles: ProfileOption[];
	    enable_adobe_dng_converter: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Setting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.worker_nums = this.convertValues(source["worker_nums"], WorkerNumOption);
	        this.profiles = this.convertValues(source["profiles"], ProfileOption);
	        this.enable_adobe_dng_converter = source["enable_adobe_dng_converter"];
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

}

