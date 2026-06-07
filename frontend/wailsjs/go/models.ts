export namespace config {
	
	export class PHPSettings {
	    processes_per_version: number;
	    processes: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new PHPSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.processes_per_version = source["processes_per_version"];
	        this.processes = source["processes"];
	    }
	}
	export class LastServiceState {
	    caddy: boolean;
	    mariadb: boolean;
	    mailpit: boolean;
	    php: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new LastServiceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.caddy = source["caddy"];
	        this.mariadb = source["mariadb"];
	        this.mailpit = source["mailpit"];
	        this.php = source["php"];
	    }
	}
	export class GlobalConfig {
	    default_www: string;
	    default_ssl: string;
	    log_file: string;
	    dependency_url: string;
	    restore_last_state: boolean;
	    minimize_to_tray: boolean;
	    run_on_boot: boolean;
	    theme: string;
	    language: string;
	    last_service_state?: LastServiceState;
	    log_level: string;
	    log_to_console: boolean;
	    max_log_retention: number;
	    max_log_lines: number;
	    auto_update_hosts: boolean;
	    terminal_shell: string;
	    php?: PHPSettings;
	    mariadb_external: boolean;
	    mariadb_basedir: string;
	    mariadb_datadir: string;
	    mariadb_type: string;
	    mariadb_port: number;
	    mariadb_user: string;
	    mariadb_password: string;
	    mailpit_smtp_port?: number;
	    mailpit_http_port?: number;
	    mailpit_use_db?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GlobalConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.default_www = source["default_www"];
	        this.default_ssl = source["default_ssl"];
	        this.log_file = source["log_file"];
	        this.dependency_url = source["dependency_url"];
	        this.restore_last_state = source["restore_last_state"];
	        this.minimize_to_tray = source["minimize_to_tray"];
	        this.run_on_boot = source["run_on_boot"];
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.last_service_state = this.convertValues(source["last_service_state"], LastServiceState);
	        this.log_level = source["log_level"];
	        this.log_to_console = source["log_to_console"];
	        this.max_log_retention = source["max_log_retention"];
	        this.max_log_lines = source["max_log_lines"];
	        this.auto_update_hosts = source["auto_update_hosts"];
	        this.terminal_shell = source["terminal_shell"];
	        this.php = this.convertValues(source["php"], PHPSettings);
	        this.mariadb_external = source["mariadb_external"];
	        this.mariadb_basedir = source["mariadb_basedir"];
	        this.mariadb_datadir = source["mariadb_datadir"];
	        this.mariadb_type = source["mariadb_type"];
	        this.mariadb_port = source["mariadb_port"];
	        this.mariadb_user = source["mariadb_user"];
	        this.mariadb_password = source["mariadb_password"];
	        this.mailpit_smtp_port = source["mailpit_smtp_port"];
	        this.mailpit_http_port = source["mailpit_http_port"];
	        this.mailpit_use_db = source["mailpit_use_db"];
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
	
	
	export class ProjectConfig {
	    id?: string;
	    name: string;
	    domains: string[];
	    type?: string;
	    runtime_type?: string;
	    php_version: string;
	    root_path: string;
	    ssl_crt: string;
	    ssl_key: string;
	    use_ssl: boolean;
	    enabled: boolean;
	    runtime_port?: number;
	    runtime_mode?: string;
	    runtime_version?: string;
	    command?: string;
	    command_dirty?: boolean;
	    use_wincmp_bin?: boolean;
	    use_env_bin?: boolean;
	    node_port?: number;
	    node_mode?: string;
	    node_version?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.domains = source["domains"];
	        this.type = source["type"];
	        this.runtime_type = source["runtime_type"];
	        this.php_version = source["php_version"];
	        this.root_path = source["root_path"];
	        this.ssl_crt = source["ssl_crt"];
	        this.ssl_key = source["ssl_key"];
	        this.use_ssl = source["use_ssl"];
	        this.enabled = source["enabled"];
	        this.runtime_port = source["runtime_port"];
	        this.runtime_mode = source["runtime_mode"];
	        this.runtime_version = source["runtime_version"];
	        this.command = source["command"];
	        this.command_dirty = source["command_dirty"];
	        this.use_wincmp_bin = source["use_wincmp_bin"];
	        this.use_env_bin = source["use_env_bin"];
	        this.node_port = source["node_port"];
	        this.node_mode = source["node_mode"];
	        this.node_version = source["node_version"];
	    }
	}
	export class WincmpConfig {
	    global: GlobalConfig;
	    projects: ProjectConfig[];
	
	    static createFrom(source: any = {}) {
	        return new WincmpConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.global = this.convertValues(source["global"], GlobalConfig);
	        this.projects = this.convertValues(source["projects"], ProjectConfig);
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

export namespace main {
	
	export class LogEntry {
	    text: string;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.time = source["time"];
	    }
	}
	export class ProjectDetectResult {
	    name: string;
	    domains: string[];
	    type: string;
	    runtime_type: string;
	    runtime_port: number;
	    php_version: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectDetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.domains = source["domains"];
	        this.type = source["type"];
	        this.runtime_type = source["runtime_type"];
	        this.runtime_port = source["runtime_port"];
	        this.php_version = source["php_version"];
	    }
	}

}

export namespace resource {
	
	export class ServiceResource {
	    name: string;
	    cpu: number;
	    ram: number;
	    pids: number[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceResource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cpu = source["cpu"];
	        this.ram = source["ram"];
	        this.pids = source["pids"];
	    }
	}
	export class ProcessResource {
	    cpu: number;
	    ram: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessResource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.ram = source["ram"];
	    }
	}
	export class DetailedResources {
	    systemCpu: number;
	    core: ProcessResource;
	    web: ProcessResource;
	    services: Record<string, ServiceResource>;
	
	    static createFrom(source: any = {}) {
	        return new DetailedResources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.systemCpu = source["systemCpu"];
	        this.core = this.convertValues(source["core"], ProcessResource);
	        this.web = this.convertValues(source["web"], ProcessResource);
	        this.services = this.convertValues(source["services"], ServiceResource, true);
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

export namespace scanner {
	
	export class PHPVersionInfo {
	    Version: string;
	    ExePath: string;
	    MajorMin: string;
	    PortBase: number;
	    PortCount: number;
	
	    static createFrom(source: any = {}) {
	        return new PHPVersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.ExePath = source["ExePath"];
	        this.MajorMin = source["MajorMin"];
	        this.PortBase = source["PortBase"];
	        this.PortCount = source["PortCount"];
	    }
	}
	export class ServiceInfo {
	    Name: string;
	    Version: string;
	    ExePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Version = source["Version"];
	        this.ExePath = source["ExePath"];
	    }
	}
	export class ScanResult {
	    CaddyList: ServiceInfo[];
	    ComposerList: ServiceInfo[];
	    HeidiSQLList: ServiceInfo[];
	    MariaDBList: ServiceInfo[];
	    MailpitList: ServiceInfo[];
	    NodeList: ServiceInfo[];
	    BunList: ServiceInfo[];
	    PHPList: PHPVersionInfo[];
	    SkippedPHP: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CaddyList = this.convertValues(source["CaddyList"], ServiceInfo);
	        this.ComposerList = this.convertValues(source["ComposerList"], ServiceInfo);
	        this.HeidiSQLList = this.convertValues(source["HeidiSQLList"], ServiceInfo);
	        this.MariaDBList = this.convertValues(source["MariaDBList"], ServiceInfo);
	        this.MailpitList = this.convertValues(source["MailpitList"], ServiceInfo);
	        this.NodeList = this.convertValues(source["NodeList"], ServiceInfo);
	        this.BunList = this.convertValues(source["BunList"], ServiceInfo);
	        this.PHPList = this.convertValues(source["PHPList"], PHPVersionInfo);
	        this.SkippedPHP = source["SkippedPHP"];
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

