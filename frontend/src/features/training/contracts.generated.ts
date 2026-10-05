// Generated from Go training DTOs. DO NOT EDIT.
// Regenerate: go run ./cmd/training-contracts

export const DEFAULT_SESSION_GAP_MINUTES = 20;

export type AnchorEvaluation = {
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
};

export type AnchorPoint = {
  at: number;
  score: number;
  samples: number;
};

export type AssessmentSpec = {
  id: string;
  protocolId: string;
  mainPlayCount: number;
  extraRuns: number;
};

export type AxisContrast = {
  key: string;
  a: number | null;
  b: number | null;
  delta: number | null;
  status: string;
  conditions?: Array<string>;
};

export type BenchmarkMembership = {
  benchmarkScore?: number;
  name: string;
  benchmarkId?: number;
  system?: string;
  nativeDifficulty?: string;
  category?: string;
  group?: string;
  thresholds?: Array<number>;
  ranks?: Array<string>;
};

export type Block = {
  observationInterrupted?: boolean;
  assessment?: AssessmentSpec;
  personalization?: SceneDecision;
  measurement?: MeasurementSpec;
  observations?: Array<PracticeSample>;
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
};

export type BlockDefinition = {
  assessment?: AssessmentSpec;
  personalization?: SceneDecision;
  measurement?: MeasurementSpec;
  curriculumRow?: number;
  completedBefore?: number;
  anchorScenario?: string;
  timing: TimingEstimate;
  difficultyEvidence: DifficultyEvidence;
  signature?: string;
  scenario: Scenario;
  role: string;
  budget: number;
  sourcePlayCount?: number;
  playCount: number;
  cue: string;
  benchmark?: string;
  initialTarget: number;
  initialReason: string;
};

export type BlockExecution = {
  observationInterrupted?: boolean;
  observations?: Array<PracticeSample>;
  lastCompletedAt?: number;
  target: number;
  reason: string;
  recorded: number;
  runs: number;
  best: number;
  outcome: string;
};

export type Candidate = {
  title: string;
  url: string;
  description: string;
  sharecodes: Array<string>;
};

export type CatalogAssessment = {
  fileStatus: string;
  difficultyStatus: string;
  hasDifficultyEvidence: boolean;
  hasPrecisionReference: boolean;
  hasBenchmarkReference: boolean;
  fit: string;
  samples: number;
};

export type Comparison = {
  kind: string;
  direction: string;
  task: string;
  hashA: string;
  hashB: string;
  axes: Array<AxisContrast>;
  unknown: Array<string>;
  changedMechanisms: Array<string>;
  basis?: Array<string>;
  neighborDistance: number | null;
  rankMargin: number | null;
  native?: NativeOrder;
  plannerUse: string;
};

export type Container = {
  format: string;
  version?: number;
  bodySHA256: string;
  trailerBytes?: number;
  trailerSHA256?: string;
};

export type Curriculum = {
  officialCode?: string;
  id: string;
  name: string;
  theme: string;
  tier?: string;
  source: Source;
  contentSHA256: string;
  rows: Array<CurriculumRow>;
};

export type CurriculumRow = {
  rowIndex?: number;
  completedBefore?: number;
  scenarioName: string;
  playCount: number;
  sourcePlayCount?: number;
  role?: string;
};

export type Definition = {
  kind: string;
  name: string;
  fields: Array<SCEField>;
};

export type DemandCoverage = {
  theme: string;
  key: string;
  min: number;
  max: number;
  scenes: Array<string>;
  status: string;
  conditions: Array<string>;
};

export type Descriptor = {
  schema: number;
  semanticVersion: string;
  fileSHA256: string;
  bodySHA256: string;
  containerSignature: string;
  mapSHA256?: string;
  gameVersion?: string;
  declaredTask: string;
  task: string;
  definitions?: Array<Definition>;
  targets: Array<Target>;
  helpers: Array<Target>;
  slots: Array<Slot>;
  scoringMin: number | null;
  scoringMax: number | null;
  axes: Array<RequirementAxis>;
  features: Array<Fact>;
  map?: MapDescriptor;
  controlledFingerprint?: string;
  unknown: Array<string>;
  hazards?: Array<HazardExposure>;
  influences?: Array<InfluenceEdge>;
};

export type DifficultyEvidence = {
  trend?: string;
  trendSessions?: number;
  recentScore?: number;
  windowDays?: number;
  benchmarks?: Array<BenchmarkMembership>;
  level: string;
  source: string;
  fit: string;
  samples: number;
};

export type Discovery = {
  updated: string;
  imported: number;
  candidates: Array<Candidate>;
  warnings: Array<string>;
};

export type DodgeMotion = {
  profile: string;
  axis: string;
  dwellMin: number;
  dwellMax: number;
  midpointEnvelope: MotionEnvelope | null;
  sources: Array<SCEField>;
};

export type ExecutionProgress = {
  planId: string;
  blocks: Array<BlockExecution>;
  endReason?: string;
  endedAt?: number;
  status: string;
  index: number;
  elapsed: number;
  recorded: number;
  blockElapsed: number;
  lastTick: number;
  acceptAfter: number;
  seen: Array<string>;
  remindedBlock?: number;
  remindedEnd?: boolean;
  reminder?: string;
};

export type ExposureSummary = {
  recordedSeconds: number;
  sameSceneSeconds: number;
  sameThemeSeconds: number;
  trialSeconds: number;
  trialScenarios?: Array<string>;
};

export type Fact = {
  key: string;
  value: number | null;
  text?: string;
  unit: string;
  status: string;
  sources?: Array<SCEField>;
  conditions?: Array<string>;
  unknown?: Array<string>;
};

export type FileMeasurement = {
  profile?: string;
  field: string;
  value: number;
  unit: string;
  line: number;
};

export type GenerateRequest = {
  preferences: Preferences;
};

export type HazardExposure = {
  bot: string;
  character: string;
  helper: boolean;
  mapSHA256: string;
  objectIndex: number;
  damageEvents: number;
  minSeconds: number;
  maxSeconds: number;
  status: string;
  conditions: Array<string>;
};

export type InfluenceEdge = {
  fromKind: string;
  from: string;
  toKind: string;
  to: string;
  field: string;
  line: number;
  role: string;
  activation: string;
  resolved: boolean;
};

export type LiveBlock = {
  recorded: number;
  runs: number;
  best: number;
  outcome: string;
  target: number;
  reason: string;
};

export type LivePlan = {
  endReason?: string;
  endedAt?: number;
  id: string;
  status: string;
  index: number;
  elapsed: number;
  recorded: number;
  blockElapsed: number;
  reminder: string;
  blocks: Array<LiveBlock>;
};

export type LiveState = {
  revision: number;
  initializing: boolean;
  notice: string;
  error: string;
  plan: LivePlan | null;
};

export type LocalAssessment = {
  requirements?: Descriptor;
  container?: Container;
  bodySHA256?: string;
  comparisonSchema?: number;
  familyFingerprint?: string;
  mapDataSHA256?: string;
  targetSizes?: Array<TargetSize>;
  measurements?: Array<FileMeasurement>;
  precisionComparisons?: Array<PrecisionComparison>;
  filePaths?: Array<string>;
  status: string;
  fileSHA256?: string;
  gameVersion?: string;
  observedAt?: string;
  filePath?: string;
  fields?: Array<SCEField>;
  issues?: Array<string>;
};

export type MapDescriptor = {
  sha256: string;
  format: string;
  status: string;
  counts: Record<string, number>;
  objects?: Array<MapObject>;
  issues?: Array<string>;
};

export type MapObject = {
  type: string;
  name: string;
  location: [number, number, number] | null;
  rotation: [number, number, number] | null;
  scale: [number, number, number] | null;
  properties?: Array<MapProperty>;
};

export type MapProperty = {
  name: string;
  raw: string;
};

export type MeasurementResult = {
  protocolId?: string;
  contextKey?: string;
  runIds?: Array<string>;
  scores?: Array<number>;
  score: number;
  accuracy?: number;
  hitsPerSecond?: number;
  signature: string;
  settings: string;
  fileSHA256: string;
  at: number;
  samples: number;
};

export type MeasurementSpec = {
  studyId: string;
  phase: string;
  protocolId?: string;
};

export type Mechanics = {
  fileSHA256: string;
  declaredSkill?: string;
  declaredSeconds?: number;
  tags: Array<string>;
  status: string;
  role: string;
  geometryStatus: string;
  angularSize: number | null;
  transitionAngle: number | null;
};

export type MotionEnvelope = {
  peakSpeed: number;
  rmsSpeed: number;
  centerExcursion: number;
  capReached: boolean;
  status: string;
  conditions: Array<string>;
};

export type NativeOrder = {
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
};

export type PersonalAnchor = {
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
  points?: Array<AnchorPoint>;
};

export type Plan = {
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
  endReason?: string;
  endedAt?: number;
  id: string;
  created: string;
  preferences: Preferences;
  blocks: Array<Block>;
  warnings: Array<string>;
  status: string;
  index: number;
  elapsed: number;
  recorded: number;
  blockElapsed: number;
  lastTick: number;
  acceptAfter: number;
  seen: Array<string>;
  remindedBlock?: number;
  remindedEnd?: boolean;
  reminder?: string;
};

export type PlanDefinition = {
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
  id: string;
  created: string;
  preferences: Preferences;
  blocks: Array<BlockDefinition>;
  warnings: Array<string>;
};

export type PlanHistorySummary = {
  id: string;
  created: string;
  status: string;
  endReason?: string;
  endedAt?: number;
  blockCount: number;
  completedBlocks: number;
  processedBlocks: number;
  runs: number;
  targetRuns: number;
  minutes: number;
  elapsed: number;
  recorded: number;
};

export type PlayerLevel = {
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
};

export type PracticeSample = {
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
};

export type PrecisionComparison = {
  reference: string;
  referenceHash: string;
  profile: string;
  radiusRatio: number;
  precisionDelta: number;
};

export type PrecisionRelation = {
  familyFingerprint: string;
  profiles: Array<PrecisionComparison>;
  direction: string;
  maxDelta: number;
  uniform: boolean;
  uniformDelta?: number;
};

export type Preferences = {
  planningPolicy?: string;
  curriculumId?: string;
  minutes: number;
  executionMode: string;
  focus: string;
  difficulty: string;
  benchmark: string;
  benchmarks?: Array<string>;
  variety: number;
  thresholdRatio: number;
  autoAdvance: boolean;
  autoDiscover: boolean;
};

export type ProgressionBudget = {
  challengeLimit: number;
  explorationLimit: number;
  unknownLimit: number;
};

export type RequirementAxis = {
  key: string;
  facts: Array<Fact>;
};

export type ResponsePrediction = {
  status: string;
  expectedScore?: number;
  samples: number;
  days: number;
  validationMAE?: number;
  baselineMAE?: number;
};

export type SCEField = {
  section: string;
  profile?: string;
  key: string;
  raw: string;
  line: number;
};

export type Scenario = {
  evaluation?: CatalogAssessment;
  name: string;
  skill: string;
  family: string;
  difficulty: string;
  difficultySource?: string;
  technique?: string;
  seconds: number;
  benchmark: string;
  thresholds?: Array<number>;
  benchmarks?: Array<BenchmarkMembership>;
  relatedBenchmarks?: Array<string>;
  variantOf?: string;
  preference?: string;
  personalDifficulty?: string;
  sources: Array<Source>;
  classification: string;
  enabled: boolean;
  mechanics?: Mechanics;
  localAssessment?: LocalAssessment;
};

export type ScenarioComparison = {
  anchor: string;
  candidate: string;
  result: Comparison;
  precision?: PrecisionRelation;
};

export type SceneDecision = {
  source: string;
  anchor: PersonalAnchor;
  relation?: PrecisionRelation;
  prediction?: ResponsePrediction;
  requirements?: ScenarioComparison;
};

export type SessionGroup = {
  id: string;
  startedAt: number;
  endedAt: number;
  runIds: Array<string>;
  legacyIds: Array<string>;
};

export type SkillStatus = {
  skill: string;
  priority: number;
  minutes: number;
  samples: number;
  evidence: string;
};

export type Slot = {
  reference: string;
  candidates: Array<string>;
  complete: boolean;
  scoringMin: number | null;
  scoringMax: number | null;
};

export type Source = {
  url: string;
  title: string;
  retrieved: string;
};

export type Target = {
  bot: string;
  character: string;
  scoring: string;
  dodgeGate: string;
  shape: string;
  facts: Array<Fact>;
  dodgeEntries?: Array<Definition>;
  abilities?: Array<Definition>;
  windows?: Array<Fact>;
  motionModels?: Array<DodgeMotion>;
};

export type TargetSize = {
  profile: string;
  radius: number;
  height?: number;
};

export type ThemePriority = {
  priority: number;
  minutes: number;
  level?: number;
  evidence: string;
};

export type TimingEstimate = {
  seconds: number;
  source: string;
  samples: number;
  recentSeconds: number;
  weeklySeconds: number;
};

export type TrainingProgressDTO = {
  current: PlanHistorySummary | null;
  recentPlans: Array<PlanHistorySummary>;
};

export type TrainingStudy = {
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
  relation: PrecisionRelation;
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
};

export type WorkbenchDTO = {
  demandCoverage?: Array<DemandCoverage>;
  version: number;
  revision: number;
  catalog: Array<Scenario>;
  preferences: Preferences;
  plan: Plan | null;
  skills: Array<SkillStatus>;
  discovery: Discovery;
  playerLevels: Array<PlayerLevel>;
  personalAnchors?: Array<PersonalAnchor>;
  anchorEvaluations?: Array<AnchorEvaluation>;
  trainingStudies?: Array<TrainingStudy>;
  themePriorities?: Record<string, ThemePriority>;
  templateTiers?: Record<string, string>;
  curricula?: Array<Curriculum>;
  initializing: boolean;
  notice: string;
  error: string;
  searchConfigured: boolean;
  recentPlans: Array<PlanHistorySummary>;
};
