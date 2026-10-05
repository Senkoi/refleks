export namespace models {

	export class BenchmarkSubcategory {
	    subcategoryName: string;
	    scenarioCount: number;
	    color?: string;

	    static createFrom(source: any = {}) {
	        return new BenchmarkSubcategory(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subcategoryName = source["subcategoryName"];
	        this.scenarioCount = source["scenarioCount"];
	        this.color = source["color"];
	    }
	}
	export class BenchmarkCategory {
	    categoryName: string;
	    color?: string;
	    subcategories: BenchmarkSubcategory[];

	    static createFrom(source: any = {}) {
	        return new BenchmarkCategory(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryName = source["categoryName"];
	        this.color = source["color"];
	        this.subcategories = this.convertValues(source["subcategories"], BenchmarkSubcategory);
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
	export class RankDef {
	    name: string;
	    color: string;

	    static createFrom(source: any = {}) {
	        return new RankDef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class BenchmarkDifficulty {
	    difficultyName: string;
	    kovaaksBenchmarkId: number;
	    sharecode: string;
	    ranks: RankDef[];
	    categories: BenchmarkCategory[];

	    static createFrom(source: any = {}) {
	        return new BenchmarkDifficulty(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.difficultyName = source["difficultyName"];
	        this.kovaaksBenchmarkId = source["kovaaksBenchmarkId"];
	        this.sharecode = source["sharecode"];
	        this.ranks = this.convertValues(source["ranks"], RankDef);
	        this.categories = this.convertValues(source["categories"], BenchmarkCategory);
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
	export class Benchmark {
	    benchmarkName: string;
	    rankCalculation: string;
	    abbreviation: string;
	    color: string;
	    spreadsheetURL: string;
	    dateAdded?: string;
	    difficulties: BenchmarkDifficulty[];

	    static createFrom(source: any = {}) {
	        return new Benchmark(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmarkName = source["benchmarkName"];
	        this.rankCalculation = source["rankCalculation"];
	        this.abbreviation = source["abbreviation"];
	        this.color = source["color"];
	        this.spreadsheetURL = source["spreadsheetURL"];
	        this.dateAdded = source["dateAdded"];
	        this.difficulties = this.convertValues(source["difficulties"], BenchmarkDifficulty);
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


	export class ScenarioProgress {
	    name: string;
	    score: number;
	    scenarioRank: number;
	    thresholds: number[];
	    energy?: number;
	    progress: number;

	    static createFrom(source: any = {}) {
	        return new ScenarioProgress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.score = source["score"];
	        this.scenarioRank = source["scenarioRank"];
	        this.thresholds = source["thresholds"];
	        this.energy = source["energy"];
	        this.progress = source["progress"];
	    }
	}
	export class ProgressGroup {
	    name?: string;
	    color?: string;
	    scenarios: ScenarioProgress[];
	    energy?: number;

	    static createFrom(source: any = {}) {
	        return new ProgressGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.scenarios = this.convertValues(source["scenarios"], ScenarioProgress);
	        this.energy = source["energy"];
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
	export class ProgressCategory {
	    name: string;
	    color?: string;
	    groups: ProgressGroup[];

	    static createFrom(source: any = {}) {
	        return new ProgressCategory(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.groups = this.convertValues(source["groups"], ProgressGroup);
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
	export class BenchmarkProgress {
	    overallRank: number;
	    benchmarkProgress: number;
	    ranks: RankDef[];
	    categories: ProgressCategory[];

	    static createFrom(source: any = {}) {
	        return new BenchmarkProgress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overallRank = source["overallRank"];
	        this.benchmarkProgress = source["benchmarkProgress"];
	        this.ranks = this.convertValues(source["ranks"], RankDef);
	        this.categories = this.convertValues(source["categories"], ProgressCategory);
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

	export class ChallengeProfileSnapshot {
	    timeLimit: number;
	    playerProfile: string;
	    addedBots: string[];
	    playerMaxLives: number;
	    botMaxLives: number[];
	    playerTeam: number;
	    botTeams: number[];
	    mapName: string;
	    mapScale: number;
	    timescale: number;
	    endChallengeAfterKills: number;
	    endChallengeAfterDamage: number;

	    static createFrom(source: any = {}) {
	        return new ChallengeProfileSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeLimit = source["timeLimit"];
	        this.playerProfile = source["playerProfile"];
	        this.addedBots = source["addedBots"];
	        this.playerMaxLives = source["playerMaxLives"];
	        this.botMaxLives = source["botMaxLives"];
	        this.playerTeam = source["playerTeam"];
	        this.botTeams = source["botTeams"];
	        this.mapName = source["mapName"];
	        this.mapScale = source["mapScale"];
	        this.timescale = source["timescale"];
	        this.endChallengeAfterKills = source["endChallengeAfterKills"];
	        this.endChallengeAfterDamage = source["endChallengeAfterDamage"];
	    }
	}
	export class KovaaksScoreAttributes {
	    fov: number;
	    hash: string;
	    cm360: number;
	    kills: number;
	    score: number;
	    avgFps: number;
	    avgTtk: number;
	    fovScale: string;
	    vertSens: number;
	    horizSens: number;
	    resolution: string;
	    sensScale: string;
	    pauseCount: number;
	    pauseDuration: number;
	    accuracyDamage: number;
	    challengeStart: string;
	    scenarioVersion: string;
	    clientBuildVersion: string;
	    epoch: string;

	    static createFrom(source: any = {}) {
	        return new KovaaksScoreAttributes(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fov = source["fov"];
	        this.hash = source["hash"];
	        this.cm360 = source["cm360"];
	        this.kills = source["kills"];
	        this.score = source["score"];
	        this.avgFps = source["avgFps"];
	        this.avgTtk = source["avgTtk"];
	        this.fovScale = source["fovScale"];
	        this.vertSens = source["vertSens"];
	        this.horizSens = source["horizSens"];
	        this.resolution = source["resolution"];
	        this.sensScale = source["sensScale"];
	        this.pauseCount = source["pauseCount"];
	        this.pauseDuration = source["pauseDuration"];
	        this.accuracyDamage = source["accuracyDamage"];
	        this.challengeStart = source["challengeStart"];
	        this.scenarioVersion = source["scenarioVersion"];
	        this.clientBuildVersion = source["clientBuildVersion"];
	        this.epoch = source["epoch"];
	    }
	}
	export class KovaaksLastScore {
	    id: string;
	    type: string;
	    attributes: KovaaksScoreAttributes;

	    static createFrom(source: any = {}) {
	        return new KovaaksLastScore(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.attributes = this.convertValues(source["attributes"], KovaaksScoreAttributes);
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




	export class ReplayStatus {
	    state: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ReplayStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	    }
	}
	export class RunEnvironment {
	    appVersion: string;
	    os: string;
	    arch: string;
	    osVersion: string;
	    steamId: string;
	    personaName: string;
	    cpuName: string;
	    cpuCores: number;
	    gpuName: string;
	    ramTotalMB: number;
	    displayHz: number;
	    screenWidth: number;
	    screenHeight: number;
	    isWindowed: boolean;
	    mouseName: string;
	    mouseVid: string;
	    mousePid: string;
	    mouseMi: string;
	    mouseBackend: string;
	    tracePoints: number;
	    traceDuration: number;
	    sampleRate: number;

	    static createFrom(source: any = {}) {
	        return new RunEnvironment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appVersion = source["appVersion"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.osVersion = source["osVersion"];
	        this.steamId = source["steamId"];
	        this.personaName = source["personaName"];
	        this.cpuName = source["cpuName"];
	        this.cpuCores = source["cpuCores"];
	        this.gpuName = source["gpuName"];
	        this.ramTotalMB = source["ramTotalMB"];
	        this.displayHz = source["displayHz"];
	        this.screenWidth = source["screenWidth"];
	        this.screenHeight = source["screenHeight"];
	        this.isWindowed = source["isWindowed"];
	        this.mouseName = source["mouseName"];
	        this.mouseVid = source["mouseVid"];
	        this.mousePid = source["mousePid"];
	        this.mouseMi = source["mouseMi"];
	        this.mouseBackend = source["mouseBackend"];
	        this.tracePoints = source["tracePoints"];
	        this.traceDuration = source["traceDuration"];
	        this.sampleRate = source["sampleRate"];
	    }
	}
	export class RunPerformanceEvent {
	    timestamp: number;
	    payloadType: string;
	    count?: number;
	    delta?: number;
	    value?: number;

	    static createFrom(source: any = {}) {
	        return new RunPerformanceEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.payloadType = source["payloadType"];
	        this.count = source["count"];
	        this.delta = source["delta"];
	        this.value = source["value"];
	    }
	}
	export class RunPerformanceHeader {
	    scenarioName: string;
	    scenarioHash: string;
	    challengeStartUtc: number;
	    schemaVersion: number;
	    challengeProfile: ChallengeProfileSnapshot;

	    static createFrom(source: any = {}) {
	        return new RunPerformanceHeader(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scenarioName = source["scenarioName"];
	        this.scenarioHash = source["scenarioHash"];
	        this.challengeStartUtc = source["challengeStartUtc"];
	        this.schemaVersion = source["schemaVersion"];
	        this.challengeProfile = this.convertValues(source["challengeProfile"], ChallengeProfileSnapshot);
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
	export class RunPerformanceData {
	    header: RunPerformanceHeader;
	    events?: RunPerformanceEvent[];

	    static createFrom(source: any = {}) {
	        return new RunPerformanceData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.header = this.convertValues(source["header"], RunPerformanceHeader);
	        this.events = this.convertValues(source["events"], RunPerformanceEvent);
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


	export class RunStatsEvent {
	    killIndex: number;
	    timestamp: string;
	    bot: string;
	    weapon: string;
	    ttkSeconds: number;
	    shots: number;
	    hits: number;
	    accuracy: number;
	    damageDone: number;
	    damagePossible: number;
	    efficiency: number;
	    cheated: boolean;
	    overShots: number;

	    static createFrom(source: any = {}) {
	        return new RunStatsEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.killIndex = source["killIndex"];
	        this.timestamp = source["timestamp"];
	        this.bot = source["bot"];
	        this.weapon = source["weapon"];
	        this.ttkSeconds = source["ttkSeconds"];
	        this.shots = source["shots"];
	        this.hits = source["hits"];
	        this.accuracy = source["accuracy"];
	        this.damageDone = source["damageDone"];
	        this.damagePossible = source["damagePossible"];
	        this.efficiency = source["efficiency"];
	        this.cheated = source["cheated"];
	        this.overShots = source["overShots"];
	    }
	}
	export class RunStatsSummary {
	    score: number;
	    kills: number;
	    deaths: number;
	    fightTime: number;
	    timeRemaining: number;
	    avgTtk: number;
	    damageDone: number;
	    totalOvershots: number;
	    damageTaken: number;
	    hitCount: number;
	    missCount: number;
	    midairs: number;
	    midaired: number;
	    directs: number;
	    directed: number;
	    reloads: number;
	    distanceTraveled: number;
	    mbsPoints: number;
	    scenario: string;
	    hash: string;
	    gameVersion: string;
	    challengeStart: string;
	    pauseCount: number;
	    pauseDuration: number;
	    avgTargetScale: number;
	    avgTimeDilation: number;
	    inputLag: number;
	    maxFpsConfig: number;
	    sensScale: string;
	    sensIncrement: number;
	    horizSens: number;
	    vertSens: number;
	    dpi: number;
	    fov: number;
	    fovScale: string;
	    hideGun: boolean;
	    crosshair: string;
	    crosshairScale: number;
	    crosshairColor: string;
	    resolution: string;
	    avgFps: number;
	    resolutionScale: number;
	    datePlayed: string;
	    accuracy: number;
	    realAvgTtk: number;
	    cm360: number;
	    duration: number;
	    scenarioTime: number;
	    time: number;

	    static createFrom(source: any = {}) {
	        return new RunStatsSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.kills = source["kills"];
	        this.deaths = source["deaths"];
	        this.fightTime = source["fightTime"];
	        this.timeRemaining = source["timeRemaining"];
	        this.avgTtk = source["avgTtk"];
	        this.damageDone = source["damageDone"];
	        this.totalOvershots = source["totalOvershots"];
	        this.damageTaken = source["damageTaken"];
	        this.hitCount = source["hitCount"];
	        this.missCount = source["missCount"];
	        this.midairs = source["midairs"];
	        this.midaired = source["midaired"];
	        this.directs = source["directs"];
	        this.directed = source["directed"];
	        this.reloads = source["reloads"];
	        this.distanceTraveled = source["distanceTraveled"];
	        this.mbsPoints = source["mbsPoints"];
	        this.scenario = source["scenario"];
	        this.hash = source["hash"];
	        this.gameVersion = source["gameVersion"];
	        this.challengeStart = source["challengeStart"];
	        this.pauseCount = source["pauseCount"];
	        this.pauseDuration = source["pauseDuration"];
	        this.avgTargetScale = source["avgTargetScale"];
	        this.avgTimeDilation = source["avgTimeDilation"];
	        this.inputLag = source["inputLag"];
	        this.maxFpsConfig = source["maxFpsConfig"];
	        this.sensScale = source["sensScale"];
	        this.sensIncrement = source["sensIncrement"];
	        this.horizSens = source["horizSens"];
	        this.vertSens = source["vertSens"];
	        this.dpi = source["dpi"];
	        this.fov = source["fov"];
	        this.fovScale = source["fovScale"];
	        this.hideGun = source["hideGun"];
	        this.crosshair = source["crosshair"];
	        this.crosshairScale = source["crosshairScale"];
	        this.crosshairColor = source["crosshairColor"];
	        this.resolution = source["resolution"];
	        this.avgFps = source["avgFps"];
	        this.resolutionScale = source["resolutionScale"];
	        this.datePlayed = source["datePlayed"];
	        this.accuracy = source["accuracy"];
	        this.realAvgTtk = source["realAvgTtk"];
	        this.cm360 = source["cm360"];
	        this.duration = source["duration"];
	        this.scenarioTime = source["scenarioTime"];
	        this.time = source["time"];
	    }
	}
	export class RunStatsData {
	    summary: RunStatsSummary;
	    events?: RunStatsEvent[];

	    static createFrom(source: any = {}) {
	        return new RunStatsData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = this.convertValues(source["summary"], RunStatsSummary);
	        this.events = this.convertValues(source["events"], RunStatsEvent);
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
	export class RunRecord {
	    fileVersion: number;
	    filePath: string;
	    fileName: string;
	    stats: RunStatsData;
	    performances?: RunPerformanceData;
	    env: RunEnvironment;
	    screenRecording?: string;

	    static createFrom(source: any = {}) {
	        return new RunRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileVersion = source["fileVersion"];
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.stats = this.convertValues(source["stats"], RunStatsData);
	        this.performances = this.convertValues(source["performances"], RunPerformanceData);
	        this.env = this.convertValues(source["env"], RunEnvironment);
	        this.screenRecording = source["screenRecording"];
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



	export class ScenarioNote {
	    notes: string;
	    sens: string;

	    static createFrom(source: any = {}) {
	        return new ScenarioNote(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notes = source["notes"];
	        this.sens = source["sens"];
	    }
	}

	export class SessionNote {
	    name: string;
	    notes: string;

	    static createFrom(source: any = {}) {
	        return new SessionNote(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.notes = source["notes"];
	    }
	}
	export class Settings {
	    steamInstallDir: string;
	    kovaaksInstallDir: string;
	    steamIdOverride?: string;
	    personaNameOverride?: string;
	    lastSeenVersion?: string;
	    sessionGapMinutes: number;
	    recentRunsDays: number;
	    recentRunsMinCount: number;
	    theme: string;
	    font?: string;
	    scale?: string;
	    language: string;
	    favoriteBenchmarks?: string[];
	    mouseTrackingEnabled: boolean;
	    mouseBufferMinutes: number;
	    screenCaptureEnabled: boolean;
	    screenCaptureFps: number;
	    screenCaptureResolution?: string;
	    replayCleanupEnabled: boolean;
	    replayRetentionDays: number;
	    replayStorageLimitGb: number;
	    autostartEnabled: boolean;
	    anonymousEnabled: boolean;
	    runSyncAvailable: boolean;
	    runSyncEnabled: boolean;
	    scenarioNotes?: Record<string, ScenarioNote>;
	    sessionNotes?: Record<string, SessionNote>;

	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steamInstallDir = source["steamInstallDir"];
	        this.kovaaksInstallDir = source["kovaaksInstallDir"];
	        this.steamIdOverride = source["steamIdOverride"];
	        this.personaNameOverride = source["personaNameOverride"];
	        this.lastSeenVersion = source["lastSeenVersion"];
	        this.sessionGapMinutes = source["sessionGapMinutes"];
	        this.recentRunsDays = source["recentRunsDays"];
	        this.recentRunsMinCount = source["recentRunsMinCount"];
	        this.theme = source["theme"];
	        this.font = source["font"];
	        this.scale = source["scale"];
	        this.language = source["language"];
	        this.favoriteBenchmarks = source["favoriteBenchmarks"];
	        this.mouseTrackingEnabled = source["mouseTrackingEnabled"];
	        this.mouseBufferMinutes = source["mouseBufferMinutes"];
	        this.screenCaptureEnabled = source["screenCaptureEnabled"];
	        this.screenCaptureFps = source["screenCaptureFps"];
	        this.screenCaptureResolution = source["screenCaptureResolution"];
	        this.replayCleanupEnabled = source["replayCleanupEnabled"];
	        this.replayRetentionDays = source["replayRetentionDays"];
	        this.replayStorageLimitGb = source["replayStorageLimitGb"];
	        this.autostartEnabled = source["autostartEnabled"];
	        this.anonymousEnabled = source["anonymousEnabled"];
	        this.runSyncAvailable = source["runSyncAvailable"];
	        this.runSyncEnabled = source["runSyncEnabled"];
	        this.scenarioNotes = this.convertValues(source["scenarioNotes"], ScenarioNote, true);
	        this.sessionNotes = this.convertValues(source["sessionNotes"], SessionNote, true);
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
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    downloadUrl?: string;
	    releaseNotes?: string;

	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNotes = source["releaseNotes"];
	    }
	}

}

export namespace runs {

	export class ScenarioHistoryPoint {
	    at: number;
	    score: number;
	    accuracy: number;
	    version: string;
	    gameVersion: string;
	    sensScale: string;
	    sens: number;
	    vertSens: number;
	    dpi: number;
	    fov: number;
	    fovScale: string;
	    seconds: number;
	    targetScale: number;
	    timeScale: number;

	    static createFrom(source: any = {}) {
	        return new ScenarioHistoryPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.score = source["score"];
	        this.accuracy = source["accuracy"];
	        this.version = source["version"];
	        this.gameVersion = source["gameVersion"];
	        this.sensScale = source["sensScale"];
	        this.sens = source["sens"];
	        this.vertSens = source["vertSens"];
	        this.dpi = source["dpi"];
	        this.fov = source["fov"];
	        this.fovScale = source["fovScale"];
	        this.seconds = source["seconds"];
	        this.targetScale = source["targetScale"];
	        this.timeScale = source["timeScale"];
	    }
	}

}

export namespace sceneanalysis {

	export class AxisContrast {
	    key: string;
	    a?: number;
	    b?: number;
	    delta?: number;
	    status: string;
	    conditions?: string[];

	    static createFrom(source: any = {}) {
	        return new AxisContrast(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.a = source["a"];
	        this.b = source["b"];
	        this.delta = source["delta"];
	        this.status = source["status"];
	        this.conditions = source["conditions"];
	    }
	}
	export class NativeOrder {
	    benchmark: string;
	    category: string;
	    family: string;
	    version: string;
	    source: string;
	    tierA: string;
	    tierB: string;
	    indexA: number;
	    indexB: number;
	    hashA: string;
	    hashB: string;
	    verified: boolean;

	    static createFrom(source: any = {}) {
	        return new NativeOrder(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmark = source["benchmark"];
	        this.category = source["category"];
	        this.family = source["family"];
	        this.version = source["version"];
	        this.source = source["source"];
	        this.tierA = source["tierA"];
	        this.tierB = source["tierB"];
	        this.indexA = source["indexA"];
	        this.indexB = source["indexB"];
	        this.hashA = source["hashA"];
	        this.hashB = source["hashB"];
	        this.verified = source["verified"];
	    }
	}
	export class Comparison {
	    kind: string;
	    direction: string;
	    task: string;
	    hashA: string;
	    hashB: string;
	    axes: AxisContrast[];
	    unknown: string[];
	    changedMechanisms: string[];
	    basis?: string[];
	    neighborDistance?: number;
	    rankMargin?: number;
	    native?: NativeOrder;
	    plannerUse: string;

	    static createFrom(source: any = {}) {
	        return new Comparison(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.direction = source["direction"];
	        this.task = source["task"];
	        this.hashA = source["hashA"];
	        this.hashB = source["hashB"];
	        this.axes = this.convertValues(source["axes"], AxisContrast);
	        this.unknown = source["unknown"];
	        this.changedMechanisms = source["changedMechanisms"];
	        this.basis = source["basis"];
	        this.neighborDistance = source["neighborDistance"];
	        this.rankMargin = source["rankMargin"];
	        this.native = this.convertValues(source["native"], NativeOrder);
	        this.plannerUse = source["plannerUse"];
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
	export class Container {
	    format: string;
	    version?: number;
	    bodySHA256: string;
	    trailerBytes?: number;
	    trailerSHA256?: string;

	    static createFrom(source: any = {}) {
	        return new Container(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.version = source["version"];
	        this.bodySHA256 = source["bodySHA256"];
	        this.trailerBytes = source["trailerBytes"];
	        this.trailerSHA256 = source["trailerSHA256"];
	    }
	}
	export class SCEField {
	    section: string;
	    profile?: string;
	    key: string;
	    raw: string;
	    line: number;

	    static createFrom(source: any = {}) {
	        return new SCEField(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.section = source["section"];
	        this.profile = source["profile"];
	        this.key = source["key"];
	        this.raw = source["raw"];
	        this.line = source["line"];
	    }
	}
	export class Definition {
	    kind: string;
	    name: string;
	    fields: SCEField[];

	    static createFrom(source: any = {}) {
	        return new Definition(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.fields = this.convertValues(source["fields"], SCEField);
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
	export class InfluenceEdge {
	    fromKind: string;
	    from: string;
	    toKind: string;
	    to: string;
	    field: string;
	    line: number;
	    role: string;
	    activation: string;
	    resolved: boolean;

	    static createFrom(source: any = {}) {
	        return new InfluenceEdge(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromKind = source["fromKind"];
	        this.from = source["from"];
	        this.toKind = source["toKind"];
	        this.to = source["to"];
	        this.field = source["field"];
	        this.line = source["line"];
	        this.role = source["role"];
	        this.activation = source["activation"];
	        this.resolved = source["resolved"];
	    }
	}
	export class HazardExposure {
	    bot: string;
	    character: string;
	    helper: boolean;
	    mapSHA256: string;
	    objectIndex: number;
	    damageEvents: number;
	    minSeconds: number;
	    maxSeconds: number;
	    status: string;
	    conditions: string[];

	    static createFrom(source: any = {}) {
	        return new HazardExposure(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bot = source["bot"];
	        this.character = source["character"];
	        this.helper = source["helper"];
	        this.mapSHA256 = source["mapSHA256"];
	        this.objectIndex = source["objectIndex"];
	        this.damageEvents = source["damageEvents"];
	        this.minSeconds = source["minSeconds"];
	        this.maxSeconds = source["maxSeconds"];
	        this.status = source["status"];
	        this.conditions = source["conditions"];
	    }
	}
	export class MapProperty {
	    name: string;
	    raw: string;

	    static createFrom(source: any = {}) {
	        return new MapProperty(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.raw = source["raw"];
	    }
	}
	export class MapObject {
	    type: string;
	    name: string;
	    location?: number[];
	    rotation?: number[];
	    scale?: number[];
	    properties?: MapProperty[];

	    static createFrom(source: any = {}) {
	        return new MapObject(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.location = source["location"];
	        this.rotation = source["rotation"];
	        this.scale = source["scale"];
	        this.properties = this.convertValues(source["properties"], MapProperty);
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
	export class MapDescriptor {
	    sha256: string;
	    format: string;
	    status: string;
	    counts: Record<string, number>;
	    objects?: MapObject[];
	    issues?: string[];

	    static createFrom(source: any = {}) {
	        return new MapDescriptor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sha256 = source["sha256"];
	        this.format = source["format"];
	        this.status = source["status"];
	        this.counts = source["counts"];
	        this.objects = this.convertValues(source["objects"], MapObject);
	        this.issues = source["issues"];
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
	export class RequirementAxis {
	    key: string;
	    facts: Fact[];

	    static createFrom(source: any = {}) {
	        return new RequirementAxis(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.facts = this.convertValues(source["facts"], Fact);
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
	export class Slot {
	    reference: string;
	    candidates: string[];
	    complete: boolean;
	    scoringMin?: number;
	    scoringMax?: number;

	    static createFrom(source: any = {}) {
	        return new Slot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reference = source["reference"];
	        this.candidates = source["candidates"];
	        this.complete = source["complete"];
	        this.scoringMin = source["scoringMin"];
	        this.scoringMax = source["scoringMax"];
	    }
	}
	export class MotionEnvelope {
	    peakSpeed: number;
	    rmsSpeed: number;
	    centerExcursion: number;
	    capReached: boolean;
	    status: string;
	    conditions: string[];

	    static createFrom(source: any = {}) {
	        return new MotionEnvelope(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peakSpeed = source["peakSpeed"];
	        this.rmsSpeed = source["rmsSpeed"];
	        this.centerExcursion = source["centerExcursion"];
	        this.capReached = source["capReached"];
	        this.status = source["status"];
	        this.conditions = source["conditions"];
	    }
	}
	export class DodgeMotion {
	    profile: string;
	    axis: string;
	    dwellMin: number;
	    dwellMax: number;
	    midpointEnvelope?: MotionEnvelope;
	    sources: SCEField[];

	    static createFrom(source: any = {}) {
	        return new DodgeMotion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.axis = source["axis"];
	        this.dwellMin = source["dwellMin"];
	        this.dwellMax = source["dwellMax"];
	        this.midpointEnvelope = this.convertValues(source["midpointEnvelope"], MotionEnvelope);
	        this.sources = this.convertValues(source["sources"], SCEField);
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
	export class Fact {
	    key: string;
	    value?: number;
	    text?: string;
	    unit: string;
	    status: string;
	    sources?: SCEField[];
	    conditions?: string[];
	    unknown?: string[];

	    static createFrom(source: any = {}) {
	        return new Fact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.text = source["text"];
	        this.unit = source["unit"];
	        this.status = source["status"];
	        this.sources = this.convertValues(source["sources"], SCEField);
	        this.conditions = source["conditions"];
	        this.unknown = source["unknown"];
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
	export class Target {
	    bot: string;
	    character: string;
	    scoring: string;
	    dodgeGate: string;
	    shape: string;
	    facts: Fact[];
	    dodgeEntries?: Definition[];
	    abilities?: Definition[];
	    windows?: Fact[];
	    motionModels?: DodgeMotion[];

	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bot = source["bot"];
	        this.character = source["character"];
	        this.scoring = source["scoring"];
	        this.dodgeGate = source["dodgeGate"];
	        this.shape = source["shape"];
	        this.facts = this.convertValues(source["facts"], Fact);
	        this.dodgeEntries = this.convertValues(source["dodgeEntries"], Definition);
	        this.abilities = this.convertValues(source["abilities"], Definition);
	        this.windows = this.convertValues(source["windows"], Fact);
	        this.motionModels = this.convertValues(source["motionModels"], DodgeMotion);
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
	export class Descriptor {
	    schema: number;
	    semanticVersion: string;
	    fileSHA256: string;
	    bodySHA256: string;
	    containerSignature: string;
	    mapSHA256?: string;
	    gameVersion?: string;
	    declaredTask: string;
	    task: string;
	    definitions?: Definition[];
	    targets: Target[];
	    helpers: Target[];
	    slots: Slot[];
	    scoringMin?: number;
	    scoringMax?: number;
	    axes: RequirementAxis[];
	    features: Fact[];
	    map?: MapDescriptor;
	    controlledFingerprint?: string;
	    unknown: string[];
	    hazards?: HazardExposure[];
	    influences?: InfluenceEdge[];

	    static createFrom(source: any = {}) {
	        return new Descriptor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.semanticVersion = source["semanticVersion"];
	        this.fileSHA256 = source["fileSHA256"];
	        this.bodySHA256 = source["bodySHA256"];
	        this.containerSignature = source["containerSignature"];
	        this.mapSHA256 = source["mapSHA256"];
	        this.gameVersion = source["gameVersion"];
	        this.declaredTask = source["declaredTask"];
	        this.task = source["task"];
	        this.definitions = this.convertValues(source["definitions"], Definition);
	        this.targets = this.convertValues(source["targets"], Target);
	        this.helpers = this.convertValues(source["helpers"], Target);
	        this.slots = this.convertValues(source["slots"], Slot);
	        this.scoringMin = source["scoringMin"];
	        this.scoringMax = source["scoringMax"];
	        this.axes = this.convertValues(source["axes"], RequirementAxis);
	        this.features = this.convertValues(source["features"], Fact);
	        this.map = this.convertValues(source["map"], MapDescriptor);
	        this.controlledFingerprint = source["controlledFingerprint"];
	        this.unknown = source["unknown"];
	        this.hazards = this.convertValues(source["hazards"], HazardExposure);
	        this.influences = this.convertValues(source["influences"], InfluenceEdge);
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


	export class FileMeasurement {
	    profile?: string;
	    field: string;
	    value: number;
	    unit: string;
	    line: number;

	    static createFrom(source: any = {}) {
	        return new FileMeasurement(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.field = source["field"];
	        this.value = source["value"];
	        this.unit = source["unit"];
	        this.line = source["line"];
	    }
	}


	export class PrecisionComparison {
	    reference: string;
	    referenceHash: string;
	    profile: string;
	    radiusRatio: number;
	    precisionDelta: number;

	    static createFrom(source: any = {}) {
	        return new PrecisionComparison(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reference = source["reference"];
	        this.referenceHash = source["referenceHash"];
	        this.profile = source["profile"];
	        this.radiusRatio = source["radiusRatio"];
	        this.precisionDelta = source["precisionDelta"];
	    }
	}
	export class TargetSize {
	    profile: string;
	    radius: number;
	    height?: number;

	    static createFrom(source: any = {}) {
	        return new TargetSize(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.radius = source["radius"];
	        this.height = source["height"];
	    }
	}
	export class LocalAssessment {
	    requirements?: Descriptor;
	    container?: Container;
	    bodySHA256?: string;
	    comparisonSchema?: number;
	    familyFingerprint?: string;
	    mapDataSHA256?: string;
	    targetSizes?: TargetSize[];
	    measurements?: FileMeasurement[];
	    precisionComparisons?: PrecisionComparison[];
	    filePaths?: string[];
	    status: string;
	    fileSHA256?: string;
	    gameVersion?: string;
	    observedAt?: string;
	    filePath?: string;
	    fields?: SCEField[];
	    issues?: string[];

	    static createFrom(source: any = {}) {
	        return new LocalAssessment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requirements = this.convertValues(source["requirements"], Descriptor);
	        this.container = this.convertValues(source["container"], Container);
	        this.bodySHA256 = source["bodySHA256"];
	        this.comparisonSchema = source["comparisonSchema"];
	        this.familyFingerprint = source["familyFingerprint"];
	        this.mapDataSHA256 = source["mapDataSHA256"];
	        this.targetSizes = this.convertValues(source["targetSizes"], TargetSize);
	        this.measurements = this.convertValues(source["measurements"], FileMeasurement);
	        this.precisionComparisons = this.convertValues(source["precisionComparisons"], PrecisionComparison);
	        this.filePaths = source["filePaths"];
	        this.status = source["status"];
	        this.fileSHA256 = source["fileSHA256"];
	        this.gameVersion = source["gameVersion"];
	        this.observedAt = source["observedAt"];
	        this.filePath = source["filePath"];
	        this.fields = this.convertValues(source["fields"], SCEField);
	        this.issues = source["issues"];
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



	export class Mechanics {
	    fileSHA256: string;
	    declaredSkill?: string;
	    declaredSeconds?: number;
	    tags: string[];
	    status: string;
	    role: string;
	    geometryStatus: string;
	    angularSize?: number;
	    transitionAngle?: number;

	    static createFrom(source: any = {}) {
	        return new Mechanics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileSHA256 = source["fileSHA256"];
	        this.declaredSkill = source["declaredSkill"];
	        this.declaredSeconds = source["declaredSeconds"];
	        this.tags = source["tags"];
	        this.status = source["status"];
	        this.role = source["role"];
	        this.geometryStatus = source["geometryStatus"];
	        this.angularSize = source["angularSize"];
	        this.transitionAngle = source["transitionAngle"];
	    }
	}



	export class PrecisionRelation {
	    familyFingerprint: string;
	    profiles: PrecisionComparison[];
	    direction: string;
	    maxDelta: number;
	    uniform: boolean;
	    uniformDelta?: number;

	    static createFrom(source: any = {}) {
	        return new PrecisionRelation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.familyFingerprint = source["familyFingerprint"];
	        this.profiles = this.convertValues(source["profiles"], PrecisionComparison);
	        this.direction = source["direction"];
	        this.maxDelta = source["maxDelta"];
	        this.uniform = source["uniform"];
	        this.uniformDelta = source["uniformDelta"];
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

export namespace screen {

	export class CaptureStatus {
	    encoderName: string;
	    container: string;
	    isHardware: boolean;
	    available: boolean;
	    active: boolean;
	    healthy: boolean;
	    state: string;
	    message: string;
	    lastError?: string;
	    lastFrameUnixMilli?: number;

	    static createFrom(source: any = {}) {
	        return new CaptureStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.encoderName = source["encoderName"];
	        this.container = source["container"];
	        this.isHardware = source["isHardware"];
	        this.available = source["available"];
	        this.active = source["active"];
	        this.healthy = source["healthy"];
	        this.state = source["state"];
	        this.message = source["message"];
	        this.lastError = source["lastError"];
	        this.lastFrameUnixMilli = source["lastFrameUnixMilli"];
	    }
	}
	export class ReplayFileInfo {
	    width: number;
	    height: number;
	    fps: number;
	    codec: string;
	    durationSeconds: number;
	    sizeBytes: number;

	    static createFrom(source: any = {}) {
	        return new ReplayFileInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.fps = source["fps"];
	        this.codec = source["codec"];
	        this.durationSeconds = source["durationSeconds"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}

}

export namespace training {

	export class ExposureSummary {
	    recordedSeconds: number;
	    sameSceneSeconds: number;
	    sameThemeSeconds: number;
	    trialSeconds: number;
	    trialScenarios?: string[];

	    static createFrom(source: any = {}) {
	        return new ExposureSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordedSeconds = source["recordedSeconds"];
	        this.sameSceneSeconds = source["sameSceneSeconds"];
	        this.sameThemeSeconds = source["sameThemeSeconds"];
	        this.trialSeconds = source["trialSeconds"];
	        this.trialScenarios = source["trialScenarios"];
	    }
	}
	export class MeasurementResult {
	    protocolId?: string;
	    contextKey?: string;
	    runIds?: string[];
	    scores?: number[];
	    score: number;
	    accuracy?: number;
	    hitsPerSecond?: number;
	    signature: string;
	    settings: string;
	    fileSHA256: string;
	    at: number;
	    samples: number;

	    static createFrom(source: any = {}) {
	        return new MeasurementResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocolId = source["protocolId"];
	        this.contextKey = source["contextKey"];
	        this.runIds = source["runIds"];
	        this.scores = source["scores"];
	        this.score = source["score"];
	        this.accuracy = source["accuracy"];
	        this.hitsPerSecond = source["hitsPerSecond"];
	        this.signature = source["signature"];
	        this.settings = source["settings"];
	        this.fileSHA256 = source["fileSHA256"];
	        this.at = source["at"];
	        this.samples = source["samples"];
	    }
	}
	export class AnchorEvaluation {
	    id: string;
	    planId: string;
	    theme: string;
	    scenario: string;
	    fileSHA256: string;
	    protocolId: string;
	    status: string;
	    reason: string;
	    createdAt: number;
	    startedAt?: number;
	    extraRuns: number;
	    extraSeconds: number;
	    result?: MeasurementResult;
	    previous?: MeasurementResult;
	    change?: number;
	    intervalHours?: number;
	    intervalKind?: string;
	    comparableDays: number;
	    exposure: ExposureSummary;

	    static createFrom(source: any = {}) {
	        return new AnchorEvaluation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.planId = source["planId"];
	        this.theme = source["theme"];
	        this.scenario = source["scenario"];
	        this.fileSHA256 = source["fileSHA256"];
	        this.protocolId = source["protocolId"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.createdAt = source["createdAt"];
	        this.startedAt = source["startedAt"];
	        this.extraRuns = source["extraRuns"];
	        this.extraSeconds = source["extraSeconds"];
	        this.result = this.convertValues(source["result"], MeasurementResult);
	        this.previous = this.convertValues(source["previous"], MeasurementResult);
	        this.change = source["change"];
	        this.intervalHours = source["intervalHours"];
	        this.intervalKind = source["intervalKind"];
	        this.comparableDays = source["comparableDays"];
	        this.exposure = this.convertValues(source["exposure"], ExposureSummary);
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
	export class AnchorPoint {
	    at: number;
	    score: number;
	    samples: number;

	    static createFrom(source: any = {}) {
	        return new AnchorPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.score = source["score"];
	        this.samples = source["samples"];
	    }
	}
	export class AssessmentSpec {
	    id: string;
	    protocolId: string;
	    mainPlayCount: number;
	    extraRuns: number;

	    static createFrom(source: any = {}) {
	        return new AssessmentSpec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.protocolId = source["protocolId"];
	        this.mainPlayCount = source["mainPlayCount"];
	        this.extraRuns = source["extraRuns"];
	    }
	}
	export class BenchmarkMembership {
	    benchmarkScore?: number;
	    name: string;
	    benchmarkId?: number;
	    system?: string;
	    nativeDifficulty?: string;
	    category?: string;
	    group?: string;
	    thresholds?: number[];
	    ranks?: string[];

	    static createFrom(source: any = {}) {
	        return new BenchmarkMembership(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmarkScore = source["benchmarkScore"];
	        this.name = source["name"];
	        this.benchmarkId = source["benchmarkId"];
	        this.system = source["system"];
	        this.nativeDifficulty = source["nativeDifficulty"];
	        this.category = source["category"];
	        this.group = source["group"];
	        this.thresholds = source["thresholds"];
	        this.ranks = source["ranks"];
	    }
	}
	export class Source {
	    url: string;
	    title: string;
	    retrieved: string;

	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.retrieved = source["retrieved"];
	    }
	}
	export class CatalogAssessment {
	    fileStatus: string;
	    difficultyStatus: string;
	    hasDifficultyEvidence: boolean;
	    hasPrecisionReference: boolean;
	    hasBenchmarkReference: boolean;
	    fit: string;
	    samples: number;

	    static createFrom(source: any = {}) {
	        return new CatalogAssessment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileStatus = source["fileStatus"];
	        this.difficultyStatus = source["difficultyStatus"];
	        this.hasDifficultyEvidence = source["hasDifficultyEvidence"];
	        this.hasPrecisionReference = source["hasPrecisionReference"];
	        this.hasBenchmarkReference = source["hasBenchmarkReference"];
	        this.fit = source["fit"];
	        this.samples = source["samples"];
	    }
	}
	export class Scenario {
	    evaluation?: CatalogAssessment;
	    name: string;
	    skill: string;
	    family: string;
	    difficulty: string;
	    difficultySource?: string;
	    technique?: string;
	    seconds: number;
	    benchmark: string;
	    thresholds?: number[];
	    benchmarks?: BenchmarkMembership[];
	    relatedBenchmarks?: string[];
	    variantOf?: string;
	    preference?: string;
	    personalDifficulty?: string;
	    sources: Source[];
	    classification: string;
	    enabled: boolean;
	    mechanics?: sceneanalysis.Mechanics;
	    localAssessment?: sceneanalysis.LocalAssessment;

	    static createFrom(source: any = {}) {
	        return new Scenario(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evaluation = this.convertValues(source["evaluation"], CatalogAssessment);
	        this.name = source["name"];
	        this.skill = source["skill"];
	        this.family = source["family"];
	        this.difficulty = source["difficulty"];
	        this.difficultySource = source["difficultySource"];
	        this.technique = source["technique"];
	        this.seconds = source["seconds"];
	        this.benchmark = source["benchmark"];
	        this.thresholds = source["thresholds"];
	        this.benchmarks = this.convertValues(source["benchmarks"], BenchmarkMembership);
	        this.relatedBenchmarks = source["relatedBenchmarks"];
	        this.variantOf = source["variantOf"];
	        this.preference = source["preference"];
	        this.personalDifficulty = source["personalDifficulty"];
	        this.sources = this.convertValues(source["sources"], Source);
	        this.classification = source["classification"];
	        this.enabled = source["enabled"];
	        this.mechanics = this.convertValues(source["mechanics"], sceneanalysis.Mechanics);
	        this.localAssessment = this.convertValues(source["localAssessment"], sceneanalysis.LocalAssessment);
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
	export class DifficultyEvidence {
	    trend?: string;
	    trendSessions?: number;
	    recentScore?: number;
	    windowDays?: number;
	    benchmarks?: BenchmarkMembership[];
	    level: string;
	    source: string;
	    fit: string;
	    samples: number;

	    static createFrom(source: any = {}) {
	        return new DifficultyEvidence(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trend = source["trend"];
	        this.trendSessions = source["trendSessions"];
	        this.recentScore = source["recentScore"];
	        this.windowDays = source["windowDays"];
	        this.benchmarks = this.convertValues(source["benchmarks"], BenchmarkMembership);
	        this.level = source["level"];
	        this.source = source["source"];
	        this.fit = source["fit"];
	        this.samples = source["samples"];
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
	export class TimingEstimate {
	    seconds: number;
	    source: string;
	    samples: number;
	    recentSeconds: number;
	    weeklySeconds: number;

	    static createFrom(source: any = {}) {
	        return new TimingEstimate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seconds = source["seconds"];
	        this.source = source["source"];
	        this.samples = source["samples"];
	        this.recentSeconds = source["recentSeconds"];
	        this.weeklySeconds = source["weeklySeconds"];
	    }
	}
	export class PracticeSample {
	    invalid?: boolean;
	    startedAt: number;
	    runId: string;
	    at: number;
	    score: number;
	    accuracy?: number;
	    hitsPerSecond?: number;
	    signature: string;
	    settings: string;
	    fileSHA256?: string;

	    static createFrom(source: any = {}) {
	        return new PracticeSample(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invalid = source["invalid"];
	        this.startedAt = source["startedAt"];
	        this.runId = source["runId"];
	        this.at = source["at"];
	        this.score = source["score"];
	        this.accuracy = source["accuracy"];
	        this.hitsPerSecond = source["hitsPerSecond"];
	        this.signature = source["signature"];
	        this.settings = source["settings"];
	        this.fileSHA256 = source["fileSHA256"];
	    }
	}
	export class MeasurementSpec {
	    studyId: string;
	    phase: string;
	    protocolId?: string;

	    static createFrom(source: any = {}) {
	        return new MeasurementSpec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.studyId = source["studyId"];
	        this.phase = source["phase"];
	        this.protocolId = source["protocolId"];
	    }
	}
	export class ScenarioComparison {
	    anchor: string;
	    candidate: string;
	    result: sceneanalysis.Comparison;
	    precision?: sceneanalysis.PrecisionRelation;

	    static createFrom(source: any = {}) {
	        return new ScenarioComparison(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anchor = source["anchor"];
	        this.candidate = source["candidate"];
	        this.result = this.convertValues(source["result"], sceneanalysis.Comparison);
	        this.precision = this.convertValues(source["precision"], sceneanalysis.PrecisionRelation);
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
	export class ResponsePrediction {
	    status: string;
	    expectedScore?: number;
	    samples: number;
	    days: number;
	    validationMAE?: number;
	    baselineMAE?: number;

	    static createFrom(source: any = {}) {
	        return new ResponsePrediction(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.expectedScore = source["expectedScore"];
	        this.samples = source["samples"];
	        this.days = source["days"];
	        this.validationMAE = source["validationMAE"];
	        this.baselineMAE = source["baselineMAE"];
	    }
	}
	export class PersonalAnchor {
	    scenario: string;
	    theme: string;
	    status: string;
	    evidence: string;
	    signature: string;
	    fileSHA256?: string;
	    medianScore: number;
	    scoreMAD: number;
	    accuracy?: number;
	    hitsPerSecond?: number;
	    samples: number;
	    sessions: number;
	    days: number;
	    lastPlayed: number;
	    points?: AnchorPoint[];

	    static createFrom(source: any = {}) {
	        return new PersonalAnchor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scenario = source["scenario"];
	        this.theme = source["theme"];
	        this.status = source["status"];
	        this.evidence = source["evidence"];
	        this.signature = source["signature"];
	        this.fileSHA256 = source["fileSHA256"];
	        this.medianScore = source["medianScore"];
	        this.scoreMAD = source["scoreMAD"];
	        this.accuracy = source["accuracy"];
	        this.hitsPerSecond = source["hitsPerSecond"];
	        this.samples = source["samples"];
	        this.sessions = source["sessions"];
	        this.days = source["days"];
	        this.lastPlayed = source["lastPlayed"];
	        this.points = this.convertValues(source["points"], AnchorPoint);
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
	export class SceneDecision {
	    source: string;
	    anchor: PersonalAnchor;
	    relation?: sceneanalysis.PrecisionRelation;
	    prediction?: ResponsePrediction;
	    requirements?: ScenarioComparison;

	    static createFrom(source: any = {}) {
	        return new SceneDecision(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.anchor = this.convertValues(source["anchor"], PersonalAnchor);
	        this.relation = this.convertValues(source["relation"], sceneanalysis.PrecisionRelation);
	        this.prediction = this.convertValues(source["prediction"], ResponsePrediction);
	        this.requirements = this.convertValues(source["requirements"], ScenarioComparison);
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
	export class Block {
	    observationInterrupted?: boolean;
	    assessment?: AssessmentSpec;
	    personalization?: SceneDecision;
	    measurement?: MeasurementSpec;
	    observations?: PracticeSample[];
	    curriculumRow?: number;
	    completedBefore?: number;
	    lastCompletedAt?: number;
	    anchorScenario?: string;
	    timing: TimingEstimate;
	    difficultyEvidence: DifficultyEvidence;
	    signature?: string;
	    scenario: Scenario;
	    role: string;
	    budget: number;
	    sourcePlayCount?: number;
	    playCount: number;
	    target: number;
	    reason: string;
	    cue: string;
	    recorded: number;
	    benchmark?: string;
	    runs: number;
	    best: number;
	    outcome: string;

	    static createFrom(source: any = {}) {
	        return new Block(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.observationInterrupted = source["observationInterrupted"];
	        this.assessment = this.convertValues(source["assessment"], AssessmentSpec);
	        this.personalization = this.convertValues(source["personalization"], SceneDecision);
	        this.measurement = this.convertValues(source["measurement"], MeasurementSpec);
	        this.observations = this.convertValues(source["observations"], PracticeSample);
	        this.curriculumRow = source["curriculumRow"];
	        this.completedBefore = source["completedBefore"];
	        this.lastCompletedAt = source["lastCompletedAt"];
	        this.anchorScenario = source["anchorScenario"];
	        this.timing = this.convertValues(source["timing"], TimingEstimate);
	        this.difficultyEvidence = this.convertValues(source["difficultyEvidence"], DifficultyEvidence);
	        this.signature = source["signature"];
	        this.scenario = this.convertValues(source["scenario"], Scenario);
	        this.role = source["role"];
	        this.budget = source["budget"];
	        this.sourcePlayCount = source["sourcePlayCount"];
	        this.playCount = source["playCount"];
	        this.target = source["target"];
	        this.reason = source["reason"];
	        this.cue = source["cue"];
	        this.recorded = source["recorded"];
	        this.benchmark = source["benchmark"];
	        this.runs = source["runs"];
	        this.best = source["best"];
	        this.outcome = source["outcome"];
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
	export class Candidate {
	    title: string;
	    url: string;
	    description: string;
	    sharecodes: string[];

	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.sharecodes = source["sharecodes"];
	    }
	}

	export class CurriculumRow {
	    rowIndex?: number;
	    completedBefore?: number;
	    scenarioName: string;
	    playCount: number;
	    sourcePlayCount?: number;
	    role?: string;

	    static createFrom(source: any = {}) {
	        return new CurriculumRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rowIndex = source["rowIndex"];
	        this.completedBefore = source["completedBefore"];
	        this.scenarioName = source["scenarioName"];
	        this.playCount = source["playCount"];
	        this.sourcePlayCount = source["sourcePlayCount"];
	        this.role = source["role"];
	    }
	}
	export class Curriculum {
	    officialCode?: string;
	    id: string;
	    name: string;
	    theme: string;
	    tier?: string;
	    source: Source;
	    contentSHA256: string;
	    rows: CurriculumRow[];

	    static createFrom(source: any = {}) {
	        return new Curriculum(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.officialCode = source["officialCode"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.theme = source["theme"];
	        this.tier = source["tier"];
	        this.source = this.convertValues(source["source"], Source);
	        this.contentSHA256 = source["contentSHA256"];
	        this.rows = this.convertValues(source["rows"], CurriculumRow);
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

	export class DemandCoverage {
	    theme: string;
	    key: string;
	    min: number;
	    max: number;
	    scenes: string[];
	    status: string;
	    conditions: string[];

	    static createFrom(source: any = {}) {
	        return new DemandCoverage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.key = source["key"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.scenes = source["scenes"];
	        this.status = source["status"];
	        this.conditions = source["conditions"];
	    }
	}

	export class Discovery {
	    updated: string;
	    imported: number;
	    candidates: Candidate[];
	    warnings: string[];

	    static createFrom(source: any = {}) {
	        return new Discovery(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updated = source["updated"];
	        this.imported = source["imported"];
	        this.candidates = this.convertValues(source["candidates"], Candidate);
	        this.warnings = source["warnings"];
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

	export class Preferences {
	    planningPolicy?: string;
	    curriculumId?: string;
	    minutes: number;
	    executionMode: string;
	    focus: string;
	    difficulty: string;
	    benchmark: string;
	    benchmarks?: string[];
	    variety: number;
	    thresholdRatio: number;
	    autoAdvance: boolean;
	    autoDiscover: boolean;

	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planningPolicy = source["planningPolicy"];
	        this.curriculumId = source["curriculumId"];
	        this.minutes = source["minutes"];
	        this.executionMode = source["executionMode"];
	        this.focus = source["focus"];
	        this.difficulty = source["difficulty"];
	        this.benchmark = source["benchmark"];
	        this.benchmarks = source["benchmarks"];
	        this.variety = source["variety"];
	        this.thresholdRatio = source["thresholdRatio"];
	        this.autoAdvance = source["autoAdvance"];
	        this.autoDiscover = source["autoDiscover"];
	    }
	}
	export class GenerateRequest {
	    preferences: Preferences;

	    static createFrom(source: any = {}) {
	        return new GenerateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preferences = this.convertValues(source["preferences"], Preferences);
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
	export class LiveBlock {
	    recorded: number;
	    runs: number;
	    best: number;
	    outcome: string;
	    target: number;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new LiveBlock(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recorded = source["recorded"];
	        this.runs = source["runs"];
	        this.best = source["best"];
	        this.outcome = source["outcome"];
	        this.target = source["target"];
	        this.reason = source["reason"];
	    }
	}
	export class LivePlan {
	    id: string;
	    status: string;
	    index: number;
	    elapsed: number;
	    recorded: number;
	    blockElapsed: number;
	    reminder: string;
	    blocks: LiveBlock[];

	    static createFrom(source: any = {}) {
	        return new LivePlan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.index = source["index"];
	        this.elapsed = source["elapsed"];
	        this.recorded = source["recorded"];
	        this.blockElapsed = source["blockElapsed"];
	        this.reminder = source["reminder"];
	        this.blocks = this.convertValues(source["blocks"], LiveBlock);
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
	export class LiveState {
	    revision: number;
	    initializing: boolean;
	    notice: string;
	    error: string;
	    plan?: LivePlan;

	    static createFrom(source: any = {}) {
	        return new LiveState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.initializing = source["initializing"];
	        this.notice = source["notice"];
	        this.error = source["error"];
	        this.plan = this.convertValues(source["plan"], LivePlan);
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



	export class ProgressionBudget {
	    challengeLimit: number;
	    explorationLimit: number;
	    unknownLimit: number;

	    static createFrom(source: any = {}) {
	        return new ProgressionBudget(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.challengeLimit = source["challengeLimit"];
	        this.explorationLimit = source["explorationLimit"];
	        this.unknownLimit = source["unknownLimit"];
	    }
	}
	export class Plan {
	    curriculumCycle?: number;
	    selectionReason?: string;
	    progression?: ProgressionBudget;
	    tierReason?: string;
	    playerTier?: string;
	    templateTier?: string;
	    plannerVersion?: number;
	    curriculumId?: string;
	    curriculumHash?: string;
	    curriculumName?: string;
	    curriculumStart?: number;
	    curriculumEnd?: number;
	    curriculumTotal?: number;
	    theme?: string;
	    endedAt?: number;
	    id: string;
	    created: string;
	    preferences: Preferences;
	    blocks: Block[];
	    warnings: string[];
	    status: string;
	    index: number;
	    elapsed: number;
	    recorded: number;
	    blockElapsed: number;
	    lastTick: number;
	    acceptAfter: number;
	    seen: string[];
	    remindedBlock?: number;
	    remindedEnd?: boolean;
	    reminder?: string;

	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.curriculumCycle = source["curriculumCycle"];
	        this.selectionReason = source["selectionReason"];
	        this.progression = this.convertValues(source["progression"], ProgressionBudget);
	        this.tierReason = source["tierReason"];
	        this.playerTier = source["playerTier"];
	        this.templateTier = source["templateTier"];
	        this.plannerVersion = source["plannerVersion"];
	        this.curriculumId = source["curriculumId"];
	        this.curriculumHash = source["curriculumHash"];
	        this.curriculumName = source["curriculumName"];
	        this.curriculumStart = source["curriculumStart"];
	        this.curriculumEnd = source["curriculumEnd"];
	        this.curriculumTotal = source["curriculumTotal"];
	        this.theme = source["theme"];
	        this.endedAt = source["endedAt"];
	        this.id = source["id"];
	        this.created = source["created"];
	        this.preferences = this.convertValues(source["preferences"], Preferences);
	        this.blocks = this.convertValues(source["blocks"], Block);
	        this.warnings = source["warnings"];
	        this.status = source["status"];
	        this.index = source["index"];
	        this.elapsed = source["elapsed"];
	        this.recorded = source["recorded"];
	        this.blockElapsed = source["blockElapsed"];
	        this.lastTick = source["lastTick"];
	        this.acceptAfter = source["acceptAfter"];
	        this.seen = source["seen"];
	        this.remindedBlock = source["remindedBlock"];
	        this.remindedEnd = source["remindedEnd"];
	        this.reminder = source["reminder"];
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
	export class PlanHistorySummary {
	    id: string;
	    created: string;
	    status: string;
	    blockCount: number;
	    minutes: number;
	    recorded: number;

	    static createFrom(source: any = {}) {
	        return new PlanHistorySummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created = source["created"];
	        this.status = source["status"];
	        this.blockCount = source["blockCount"];
	        this.minutes = source["minutes"];
	        this.recorded = source["recorded"];
	    }
	}
	export class PlayerLevel {
	    trainingAtCeiling?: boolean;
	    trainingTier?: string;
	    ability: number;
	    atCeiling?: boolean;
	    source?: string;
	    windowDays?: number;
	    lastPlayed?: string;
	    theme: string;
	    category?: string;
	    group?: string;
	    system?: string;
	    nativeDifficulty?: string;
	    rank?: string;
	    tier: string;
	    status: string;
	    scenarios: number;
	    required: number;
	    samples: number;
	    evidence: string;

	    static createFrom(source: any = {}) {
	        return new PlayerLevel(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trainingAtCeiling = source["trainingAtCeiling"];
	        this.trainingTier = source["trainingTier"];
	        this.ability = source["ability"];
	        this.atCeiling = source["atCeiling"];
	        this.source = source["source"];
	        this.windowDays = source["windowDays"];
	        this.lastPlayed = source["lastPlayed"];
	        this.theme = source["theme"];
	        this.category = source["category"];
	        this.group = source["group"];
	        this.system = source["system"];
	        this.nativeDifficulty = source["nativeDifficulty"];
	        this.rank = source["rank"];
	        this.tier = source["tier"];
	        this.status = source["status"];
	        this.scenarios = source["scenarios"];
	        this.required = source["required"];
	        this.samples = source["samples"];
	        this.evidence = source["evidence"];
	    }
	}







	export class SkillStatus {
	    skill: string;
	    priority: number;
	    minutes: number;
	    samples: number;
	    evidence: string;

	    static createFrom(source: any = {}) {
	        return new SkillStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skill = source["skill"];
	        this.priority = source["priority"];
	        this.minutes = source["minutes"];
	        this.samples = source["samples"];
	        this.evidence = source["evidence"];
	    }
	}

	export class ThemePriority {
	    priority: number;
	    minutes: number;
	    level?: number;
	    evidence: string;

	    static createFrom(source: any = {}) {
	        return new ThemePriority(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.priority = source["priority"];
	        this.minutes = source["minutes"];
	        this.level = source["level"];
	        this.evidence = source["evidence"];
	    }
	}

	export class TrainingStudy {
	    protocolId?: string;
	    transferContaminated?: boolean;
	    id: string;
	    planId: string;
	    theme: string;
	    anchorScenario: string;
	    trainingScenario: string;
	    transferScenario?: string;
	    anchorHash: string;
	    trainingHash: string;
	    transferHash?: string;
	    relation: sceneanalysis.PrecisionRelation;
	    status: string;
	    feedback?: string;
	    createdAt: number;
	    trainedAt?: number;
	    dueAt?: number;
	    expiresAt?: number;
	    baseline?: MeasurementResult;
	    trial?: MeasurementResult;
	    retest?: MeasurementResult;
	    transferBaseline?: MeasurementResult;
	    transferRetest?: MeasurementResult;
	    retentionChange?: number;
	    transferChange?: number;

	    static createFrom(source: any = {}) {
	        return new TrainingStudy(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocolId = source["protocolId"];
	        this.transferContaminated = source["transferContaminated"];
	        this.id = source["id"];
	        this.planId = source["planId"];
	        this.theme = source["theme"];
	        this.anchorScenario = source["anchorScenario"];
	        this.trainingScenario = source["trainingScenario"];
	        this.transferScenario = source["transferScenario"];
	        this.anchorHash = source["anchorHash"];
	        this.trainingHash = source["trainingHash"];
	        this.transferHash = source["transferHash"];
	        this.relation = this.convertValues(source["relation"], sceneanalysis.PrecisionRelation);
	        this.status = source["status"];
	        this.feedback = source["feedback"];
	        this.createdAt = source["createdAt"];
	        this.trainedAt = source["trainedAt"];
	        this.dueAt = source["dueAt"];
	        this.expiresAt = source["expiresAt"];
	        this.baseline = this.convertValues(source["baseline"], MeasurementResult);
	        this.trial = this.convertValues(source["trial"], MeasurementResult);
	        this.retest = this.convertValues(source["retest"], MeasurementResult);
	        this.transferBaseline = this.convertValues(source["transferBaseline"], MeasurementResult);
	        this.transferRetest = this.convertValues(source["transferRetest"], MeasurementResult);
	        this.retentionChange = source["retentionChange"];
	        this.transferChange = source["transferChange"];
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
	export class WorkbenchDTO {
	    demandCoverage?: DemandCoverage[];
	    version: number;
	    revision: number;
	    catalog: Scenario[];
	    preferences: Preferences;
	    plan?: Plan;
	    skills: SkillStatus[];
	    discovery: Discovery;
	    playerLevels: PlayerLevel[];
	    personalAnchors?: PersonalAnchor[];
	    anchorEvaluations?: AnchorEvaluation[];
	    trainingStudies?: TrainingStudy[];
	    themePriorities?: Record<string, ThemePriority>;
	    templateTiers?: Record<string, string>;
	    curricula?: Curriculum[];
	    initializing: boolean;
	    notice: string;
	    error: string;
	    searchConfigured: boolean;
	    recentPlans: PlanHistorySummary[];

	    static createFrom(source: any = {}) {
	        return new WorkbenchDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demandCoverage = this.convertValues(source["demandCoverage"], DemandCoverage);
	        this.version = source["version"];
	        this.revision = source["revision"];
	        this.catalog = this.convertValues(source["catalog"], Scenario);
	        this.preferences = this.convertValues(source["preferences"], Preferences);
	        this.plan = this.convertValues(source["plan"], Plan);
	        this.skills = this.convertValues(source["skills"], SkillStatus);
	        this.discovery = this.convertValues(source["discovery"], Discovery);
	        this.playerLevels = this.convertValues(source["playerLevels"], PlayerLevel);
	        this.personalAnchors = this.convertValues(source["personalAnchors"], PersonalAnchor);
	        this.anchorEvaluations = this.convertValues(source["anchorEvaluations"], AnchorEvaluation);
	        this.trainingStudies = this.convertValues(source["trainingStudies"], TrainingStudy);
	        this.themePriorities = this.convertValues(source["themePriorities"], ThemePriority, true);
	        this.templateTiers = source["templateTiers"];
	        this.curricula = this.convertValues(source["curricula"], Curriculum);
	        this.initializing = source["initializing"];
	        this.notice = source["notice"];
	        this.error = source["error"];
	        this.searchConfigured = source["searchConfigured"];
	        this.recentPlans = this.convertValues(source["recentPlans"], PlanHistorySummary);
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

