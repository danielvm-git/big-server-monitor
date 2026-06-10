export namespace activitylog {
	
	export class ActivityEvent {
	    id: number;
	    type: string;
	    port: number;
	    processName: string;
	    projectName: string;
	    projectDir: string;
	    // Go type: time
	    timestamp: any;
	    duration?: string;
	    exitCode?: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivityEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.port = source["port"];
	        this.processName = source["processName"];
	        this.projectName = source["projectName"];
	        this.projectDir = source["projectDir"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.duration = source["duration"];
	        this.exitCode = source["exitCode"];
	        this.message = source["message"];
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
	export class ActivityFilter {
	    ProjectName: string;
	    Port: number;
	    EventTypes: string[];
	    // Go type: time
	    Since: any;
	    Limit: number;
	    Offset: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProjectName = source["ProjectName"];
	        this.Port = source["Port"];
	        this.EventTypes = source["EventTypes"];
	        this.Since = this.convertValues(source["Since"], null);
	        this.Limit = source["Limit"];
	        this.Offset = source["Offset"];
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

export namespace healthcheck {
	
	export class HealthResult {
	    port: number;
	    status: string;
	    statusCode: number;
	    latencyMs: number;
	    protocol: string;
	    scheme?: string;
	    // Go type: time
	    checkedAt: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.status = source["status"];
	        this.statusCode = source["statusCode"];
	        this.latencyMs = source["latencyMs"];
	        this.protocol = source["protocol"];
	        this.scheme = source["scheme"];
	        this.checkedAt = this.convertValues(source["checkedAt"], null);
	        this.error = source["error"];
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

export namespace logcapture {
	
	export class LogFilter {
	    Port: number;
	    Levels: string[];
	    Limit: number;
	
	    static createFrom(source: any = {}) {
	        return new LogFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Port = source["Port"];
	        this.Levels = source["Levels"];
	        this.Limit = source["Limit"];
	    }
	}
	export class LogLine {
	    seq: number;
	    // Go type: time
	    timestamp: any;
	    level: string;
	    text: string;
	    stream: string;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.text = source["text"];
	        this.stream = source["stream"];
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

export namespace processmonitor {
	
	export class EnvVar {
	    key: string;
	    value: string;
	    visible: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.visible = source["visible"];
	    }
	}
	export class Server {
	    port: number;
	    status: string;
	    pid: number;
	    processName: string;
	    runtimeVersion: string;
	    binaryPath: string;
	    projectName: string;
	    projectDir: string;
	    localDomain?: string;
	    tunnelURL?: string;
	    envSnapshot?: EnvVar[];
	    memoryMb: number;
	    // Go type: time
	    startedAt: any;
	    uptimeStr: string;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.status = source["status"];
	        this.pid = source["pid"];
	        this.processName = source["processName"];
	        this.runtimeVersion = source["runtimeVersion"];
	        this.binaryPath = source["binaryPath"];
	        this.projectName = source["projectName"];
	        this.projectDir = source["projectDir"];
	        this.localDomain = source["localDomain"];
	        this.tunnelURL = source["tunnelURL"];
	        this.envSnapshot = this.convertValues(source["envSnapshot"], EnvVar);
	        this.memoryMb = source["memoryMb"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.uptimeStr = source["uptimeStr"];
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

export namespace settings {
	
	export class Config {
	    scanDirectories: string[];
	    pollingIntervalSeconds: number;
	    healthCheckIntervalSeconds: number;
	    ignoredPorts: number[];
	    logRetentionDays: number;
	    // Go type: struct { CrashAlerts bool "json:\"crashAlerts\""; ShowBadge bool "json:\"showBadge\"" }
	    notifications: any;
	    launchAtLogin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanDirectories = source["scanDirectories"];
	        this.pollingIntervalSeconds = source["pollingIntervalSeconds"];
	        this.healthCheckIntervalSeconds = source["healthCheckIntervalSeconds"];
	        this.ignoredPorts = source["ignoredPorts"];
	        this.logRetentionDays = source["logRetentionDays"];
	        this.notifications = this.convertValues(source["notifications"], Object);
	        this.launchAtLogin = source["launchAtLogin"];
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

