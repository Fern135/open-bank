

class Config {
    private static readonly baseUrl = "http://localhost:3001";
    private readonly appName = "Open Tally";
    private static readonly apiVersion = "v1";
    private readonly appDescription = "Checking and savings with the books open. See every entry behind your balance.";
    public  static readonly apiBaseUrl = `${this.baseUrl}/api/${this.apiVersion}/`;
    
    constructor() {
    }
    
    public getAppName(): string {
        return this.appName;
    }
    
    public getAppDescription(): string {
        return this.appDescription;
    }
}

export default new Config();