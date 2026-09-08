import { describe, expect, it, vi } from 'vitest'

import { fetchDashboard } from './api'

describe('fetchDashboard', () => {
  it('loads and validates the allowlisted dashboard snapshot with safe projections', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          strategy_version: 'bold-momentum-v2',
          utc_date: '2026-08-20',
          daily_results: [{ strategy_version: 'strategy-a', daily_admitted_count: 2, realized_pnl_micros: 1500000 }],
          open_positions: [{ id: 7, candidate_id: 9, state: 'OPEN', notional_micros: 10000000, mint_address: 'public-token-mint', no_route_count: 1, opened_at: '2026-08-20T01:00:00Z' }],
          twitter_analytics: [{ mint_address: 'twitter-public-mint', searched_at: '2026-08-20T01:00:00Z', search_count: 1, score: 72, posts: 8, unique_authors: 6, exact_mint_mentions: 5, warning_posts: 1 }],
          automation_activity: [{ occurred_at: '2026-08-20T02:00:00Z', job: 'SCAN', outcome: 'COMPLETED', category: '', stage: '' }],
          reviewed_candidates: [{ mint_address: 'reviewed-public-mint', checked_at: '2026-08-20T03:00:00Z', outcome: 'NOT_TRADED', reason: 'social_evidence_unavailable' }],
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchDashboard('2026-08-20')).resolves.toEqual({
      strategyVersion: 'bold-momentum-v2',
      utcDate: '2026-08-20',
      dailyResults: [{ strategyVersion: 'strategy-a', dailyAdmittedCount: 2, realizedPnlMicros: 1500000 }],
      openPositions: [{ id: 7, candidateId: 9, state: 'OPEN', notionalMicros: 10000000, mintAddress: 'public-token-mint', noRouteCount: 1, openedAt: '2026-08-20T01:00:00Z' }],
      twitterAnalytics: [{ mintAddress: 'twitter-public-mint', searchedAt: '2026-08-20T01:00:00Z', searchCount: 1, score: 72, posts: 8, uniqueAuthors: 6, exactMintMentions: 5, warningPosts: 1 }],
      automationActivity: [{ occurredAt: '2026-08-20T02:00:00Z', job: 'SCAN', outcome: 'COMPLETED', category: '', stage: '' }],
      reviewedCandidates: [{ mintAddress: 'reviewed-public-mint', checkedAt: '2026-08-20T03:00:00Z', outcome: 'NOT_TRADED', reason: 'social_evidence_unavailable' }],
    })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dashboard?date=2026-08-20', {
      headers: { Accept: 'application/json' },
      signal: undefined,
    })
  })

  it('rejects an out-of-range Twitter score from the API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      strategy_version: 'bold-momentum-v2', utc_date: '2026-08-20', daily_results: [], open_positions: [],
      twitter_analytics: [{ mint_address: 'twitter-public-mint', searched_at: '2026-08-20T01:00:00Z', search_count: 1, score: 101, posts: 0, unique_authors: 0, exact_mint_mentions: 0, warning_posts: 0 }],
    }), { status: 200 })))

    await expect(fetchDashboard('2026-08-20')).rejects.toThrow('Twitter analytics score must be between 0 and 100')
  })

  it.each([
    'deterministic_five_minute_buy_share',
    'deterministic_five_minute_turnover',
    'deterministic_five_minute_price_change',
    'deterministic_social_quality',
    'hermes_verdict_threshold',
  ] as const)('accepts fixed reviewed-candidate reason %s', async (reason) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      strategy_version: 'bold-momentum-v2', utc_date: '2026-08-20', daily_results: [], open_positions: [], twitter_analytics: [], automation_activity: [],
      reviewed_candidates: [{ mint_address: 'public-mint', checked_at: '2026-08-20T03:00:00Z', outcome: 'NOT_TRADED', reason }],
    }), { status: 200 })))

    const snapshot = await fetchDashboard('2026-08-20')
    expect(snapshot.reviewedCandidates[0].reason).toBe(reason)
  })

  it('rejects arbitrary provider errors as reviewed-candidate reasons', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      strategy_version: 'bold-momentum-v2', utc_date: '2026-08-20', daily_results: [], open_positions: [], twitter_analytics: [], automation_activity: [],
      reviewed_candidates: [{ mint_address: 'public-mint', checked_at: '2026-08-20T03:00:00Z', outcome: 'NOT_TRADED', reason: 'provider returned api-key=secret' }],
    }), { status: 200 })))

    await expect(fetchDashboard('2026-08-20')).rejects.toThrow('reviewed candidate reason is unsupported')
  })
})
