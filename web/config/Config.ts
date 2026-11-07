

class Config {
    private readonly appName = "Open Tally";
    private static readonly apiVersion = "v1";
    private readonly appDescription = "Checking and savings with the books open. See every entry behind your balance.";
    // Same origin on purpose: next.config.ts proxies this path to the Go server,
    // which lets the auth cookies be HttpOnly and SameSite=Strict.
    private static readonly apiBaseUrl = `/api/${this.apiVersion}/`;
    
    constructor() {
    }
    
    public getApiBaseUrl(): string {
        return Config.apiBaseUrl;
    }

    public getAppName(): string {
        return this.appName;
    }
    
    public getAppDescription(): string {
        return this.appDescription;
    }
}

export default new Config();