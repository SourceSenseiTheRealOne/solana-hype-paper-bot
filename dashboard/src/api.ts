export type DailyResult = {
  strategyVersion: string
  dailyAdmittedCount: number
  realizedPnlMicros: number
}

export type OpenPosition = {
  id: number
  candidateId: number
  state: string
  notionalMicros: number
  mintAddress: string
  noRouteCount: number
  openedAt: string
}

export type TwitterAnalytics = {
  mintAddress: string
  searchedAt: string
  searchCount: number
  score: number
  posts: number
  uniqueAuthors: number
  exactMintMentions: number
  warningPosts: number
}

export type AutomationActivity = {
  occurredAt: string
  job: 'SCAN' | 'MONITOR'
  outcome: 'STARTED' | 'COMPLETED' | 'FAILED'
  category: string
  stage: string
}

export type ReviewedCandidate = {
  mintAddress: string
  checkedAt: string
  outcome: 'NOT_TRADED'
  reason: ReviewedCandidateReason
}

export type ReviewedCandidateReason =
  | 'deterministic_pool_age'
  | 'deterministic_liquidity'
  | 'deterministic_five_minute_activity'
  | 'deterministic_five_minute_buy_share'
  | 'deterministic_five_minute_turnover'
  | 'deterministic_five_minute_price_change'
  | 'market_evidence_unavailable'
  | 'social_evidence_unavailable'
  | 'deterministic_social_quality'
  | 'hermes_verdict_threshold'
  | 'deterministic_entry_price_impact'
  | 'deterministic_entry_route'
  | 'deterministic_exit_route'
  | 'deterministic_mint_authority_revoked'
  | 'deterministic_freeze_authority_revoked'
  | 'deterministic_token_program_extensions'
  | 'admission_not_admitted'
  | 'legacy_reason_unavailable'

export type DashboardSnapshot = {
  strategyVersion: string
  utcDate: string
  dailyResults: DailyResult[]
  openPositions: OpenPosition[]
  twitterAnalytics: TwitterAnalytics[]
  automationActivity: AutomationActivity[]
  reviewedCandidates: ReviewedCandidate[]
}

type JsonRecord = Record<string, unknown>

export async function fetchDashboard(date: string, signal?: AbortSignal): Promise<DashboardSnapshot> {
  if (!isUTCDate(date)) {
    throw new Error('dashboard date must be YYYY-MM-DD')
  }

  const response = await fetch(`/api/v1/dashboard?date=${encodeURIComponent(date)}`, {
    headers: { Accept: 'application/json' },
    signal,
  })
  if (!response.ok) {
    throw new Error('dashboard data is unavailable')
  }

  return dashboardFrom(await response.json())
}

function dashboardFrom(value: unknown): DashboardSnapshot {
  const record = object(value, 'dashboard response')
  const utcDate = text(record.utc_date, 'utc_date')
  if (!isUTCDate(utcDate)) {
    throw new Error('dashboard response has an invalid utc_date')
  }

  return {
    strategyVersion: text(record.strategy_version, 'strategy_version'),
    utcDate,
    dailyResults: array(record.daily_results, 'daily_results').map(dailyResultFrom),
    openPositions: array(record.open_positions, 'open_positions').map(openPositionFrom),
    twitterAnalytics: boundedArray(record.twitter_analytics, 'twitter_analytics').map(twitterAnalyticsFrom),
    automationActivity: boundedArray(record.automation_activity, 'automation_activity').map(automationActivityFrom),
    reviewedCandidates: boundedArray(record.reviewed_candidates, 'reviewed_candidates').map(reviewedCandidateFrom),
  }
}

function dailyResultFrom(value: unknown): DailyResult {
  const record = object(value, 'daily result')
  return {
    strategyVersion: text(record.strategy_version, 'daily result strategy_version'),
    dailyAdmittedCount: nonNegativeInteger(record.daily_admitted_count, 'daily result daily_admitted_count'),
    realizedPnlMicros: safeInteger(record.realized_pnl_micros, 'daily result realized_pnl_micros'),
  }
}

function openPositionFrom(value: unknown): OpenPosition {
  const record = object(value, 'open position')
  const openedAt = text(record.opened_at, 'open position opened_at')
  if (openedAt !== '' && Number.isNaN(Date.parse(openedAt))) {
    throw new Error('open position opened_at must be ISO-8601 or empty')
  }

  return {
    id: positiveInteger(record.id, 'open position id'),
    candidateId: positiveInteger(record.candidate_id, 'open position candidate_id'),
    state: text(record.state, 'open position state'),
    notionalMicros: nonNegativeInteger(record.notional_micros, 'open position notional_micros'),
    mintAddress: text(record.mint_address, 'open position mint_address'),
    noRouteCount: nonNegativeInteger(record.no_route_count, 'open position no_route_count'),
    openedAt,
  }
}

function twitterAnalyticsFrom(value: unknown): TwitterAnalytics {
  const record = object(value, 'Twitter analytics')
  return {
    mintAddress: text(record.mint_address, 'Twitter analytics mint_address'),
    searchedAt: utcTimestamp(record.searched_at, 'Twitter analytics searched_at'),
    searchCount: exactlyOne(record.search_count, 'Twitter analytics search_count'),
    score: twitterScore(record.score),
    posts: nonNegativeInteger(record.posts, 'Twitter analytics posts'),
    uniqueAuthors: nonNegativeInteger(record.unique_authors, 'Twitter analytics unique_authors'),
    exactMintMentions: nonNegativeInteger(record.exact_mint_mentions, 'Twitter analytics exact_mint_mentions'),
    warningPosts: nonNegativeInteger(record.warning_posts, 'Twitter analytics warning_posts'),
  }
}

function automationActivityFrom(value: unknown): AutomationActivity {
  const record = object(value, 'automation activity')
  const job = text(record.job, 'automation activity job')
  const outcome = text(record.outcome, 'automation activity outcome')
  const category = text(record.category, 'automation activity category')
  const stage = text(record.stage, 'automation activity stage')
  if (job !== 'SCAN' && job !== 'MONITOR') {
    throw new Error('automation activity job is unsupported')
  }
  if (outcome !== 'STARTED' && outcome !== 'COMPLETED' && outcome !== 'FAILED') {
    throw new Error('automation activity outcome is unsupported')
  }
  if ((outcome === 'FAILED') !== (category !== '' && stage !== '')) {
    throw new Error('automation activity failure fields are invalid')
  }
  return { occurredAt: utcTimestamp(record.occurred_at, 'automation activity occurred_at'), job, outcome, category, stage }
}

function reviewedCandidateFrom(value: unknown): ReviewedCandidate {
  const record = object(value, 'reviewed candidate')
  const outcome = text(record.outcome, 'reviewed candidate outcome')
  const reason = text(record.reason, 'reviewed candidate reason')
  if (outcome !== 'NOT_TRADED') {
	throw new Error('reviewed candidate outcome is unsupported')
  }
	if (!isReviewedCandidateReason(reason)) {
	  throw new Error('reviewed candidate reason is unsupported')
	}
  return {
    mintAddress: text(record.mint_address, 'reviewed candidate mint_address'),
    checkedAt: utcTimestamp(record.checked_at, 'reviewed candidate checked_at'),
    outcome,
	  reason,
  }
}

function isReviewedCandidateReason(value: string): value is ReviewedCandidateReason {
  return [
    'deterministic_pool_age', 'deterministic_liquidity', 'deterministic_five_minute_activity', 'deterministic_five_minute_buy_share', 'deterministic_five_minute_turnover', 'deterministic_five_minute_price_change', 'market_evidence_unavailable', 'social_evidence_unavailable', 'deterministic_social_quality', 'hermes_verdict_threshold', 'deterministic_entry_price_impact', 'deterministic_entry_route', 'deterministic_exit_route', 'deterministic_mint_authority_revoked', 'deterministic_freeze_authority_revoked', 'deterministic_token_program_extensions', 'admission_not_admitted', 'legacy_reason_unavailable',
  ].includes(value as ReviewedCandidateReason)
}

function array(value: unknown, label: string): unknown[] {
  if (!Array.isArray(value)) {
    throw new Error(`${label} must be an array`)
  }
  return value
}

function boundedArray(value: unknown, label: string): unknown[] {
  const values = array(value, label)
  if (values.length > 10) {
    throw new Error(`${label} must contain at most 10 items`)
  }
  return values
}

function object(value: unknown, label: string): JsonRecord {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be an object`)
  }
  return value as JsonRecord
}

function text(value: unknown, label: string): string {
  if (typeof value !== 'string') {
    throw new Error(`${label} must be a string`)
  }
  return value
}

function utcTimestamp(value: unknown, label: string): string {
  const timestamp = text(value, label)
  if (!timestamp.endsWith('Z') || Number.isNaN(Date.parse(timestamp))) {
    throw new Error(`${label} must be an RFC3339 UTC timestamp`)
  }
  return timestamp
}

function exactlyOne(value: unknown, label: string): number {
  const integer = safeInteger(value, label)
  if (integer !== 1) {
    throw new Error(`${label} must equal 1`)
  }
  return integer
}

function twitterScore(value: unknown): number {
  const integer = safeInteger(value, 'Twitter analytics score')
  if (integer < 0 || integer > 100) {
    throw new Error('Twitter analytics score must be between 0 and 100')
  }
  return integer
}

function positiveInteger(value: unknown, label: string): number {
  const integer = safeInteger(value, label)
  if (integer <= 0) {
    throw new Error(`${label} must be positive`)
  }
  return integer
}

function nonNegativeInteger(value: unknown, label: string): number {
  const integer = safeInteger(value, label)
  if (integer < 0) {
    throw new Error(`${label} must not be negative`)
  }
  return integer
}

function safeInteger(value: unknown, label: string): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value)) {
    throw new Error(`${label} must be a safe integer`)
  }
  return value
}

function isUTCDate(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value) && !Number.isNaN(Date.parse(`${value}T00:00:00Z`))
}
