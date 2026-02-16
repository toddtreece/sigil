import {
  buildLabelSelector,
  computeStep,
  computeRateInterval,
  computeRangeDuration,
  totalOpsQuery,
  totalTokensQuery,
  totalErrorsQuery,
  errorRateQuery,
  tokenUsageOverTimeQuery,
  callsByProviderQuery,
  topModelsQuery,
  latencyP95Query,
  ttftP95Query,
  tokensByModelAndTypeQuery,
} from './queries';
import type { DashboardFilters } from './types';

describe('buildLabelSelector', () => {
  it('returns empty string for empty filters', () => {
    expect(buildLabelSelector({ provider: '', model: '', agentName: '' })).toBe('');
  });

  it('builds selector for single filter', () => {
    expect(buildLabelSelector({ provider: 'openai', model: '', agentName: '' })).toBe('gen_ai_provider_name="openai"');
  });

  it('builds selector for multiple filters', () => {
    expect(buildLabelSelector({ provider: 'openai', model: 'gpt-4o', agentName: 'my-agent' })).toBe(
      'gen_ai_provider_name="openai",gen_ai_request_model="gpt-4o",gen_ai_agent_name="my-agent"'
    );
  });
});

describe('computeStep', () => {
  it('returns minimum of 15 for short ranges', () => {
    expect(computeStep(0, 100)).toBe(15);
  });

  it('computes step for longer ranges', () => {
    // 1 hour = 3600s, 3600/250 = 14.4, min 15
    expect(computeStep(0, 3600)).toBe(15);
    // 24 hours = 86400s, 86400/250 = 345.6 → 345
    expect(computeStep(0, 86400)).toBe(345);
  });
});

describe('computeRateInterval', () => {
  it('returns at least 60s', () => {
    expect(computeRateInterval(10)).toBe('60s');
  });

  it('returns 4x step when larger than 60s', () => {
    expect(computeRateInterval(30)).toBe('120s');
  });
});

describe('computeRangeDuration', () => {
  it('returns difference as seconds string', () => {
    expect(computeRangeDuration(1000, 2000)).toBe('1000s');
  });
});

describe('query builders', () => {
  const noFilters: DashboardFilters = { provider: '', model: '', agentName: '' };
  const withFilters: DashboardFilters = { provider: 'openai', model: '', agentName: '' };

  it('totalOpsQuery without filters', () => {
    expect(totalOpsQuery(noFilters, '3600s')).toBe('sum(increase(gen_ai_client_operation_duration_count[3600s]))');
  });

  it('totalOpsQuery with filters', () => {
    expect(totalOpsQuery(withFilters, '3600s')).toBe(
      'sum(increase(gen_ai_client_operation_duration_count{gen_ai_provider_name="openai"}[3600s]))'
    );
  });

  it('totalTokensQuery', () => {
    expect(totalTokensQuery(noFilters, '3600s')).toBe('sum(increase(gen_ai_client_token_usage_sum[3600s]))');
  });

  it('totalErrorsQuery', () => {
    expect(totalErrorsQuery(noFilters, '3600s')).toBe(
      'sum(increase(gen_ai_client_operation_duration_count{error_type!=""}[3600s]))'
    );
  });

  it('errorRateQuery', () => {
    const q = errorRateQuery(noFilters, '3600s');
    expect(q).toContain('error_type!=""');
    expect(q).toContain('* 100');
  });

  it('tokenUsageOverTimeQuery', () => {
    expect(tokenUsageOverTimeQuery(noFilters, '60s')).toBe(
      'sum by (gen_ai_token_type) (rate(gen_ai_client_token_usage_sum[60s]))'
    );
  });

  it('callsByProviderQuery', () => {
    expect(callsByProviderQuery(noFilters, '3600s')).toBe(
      'sum by (gen_ai_provider_name) (increase(gen_ai_client_operation_duration_count[3600s]))'
    );
  });

  it('topModelsQuery', () => {
    expect(topModelsQuery(noFilters, '3600s')).toBe(
      'sum by (gen_ai_request_model) (increase(gen_ai_client_operation_duration_count[3600s]))'
    );
  });

  it('latencyP95Query', () => {
    expect(latencyP95Query(noFilters, '60s')).toContain('histogram_quantile(0.95');
    expect(latencyP95Query(noFilters, '60s')).toContain('gen_ai_client_operation_duration_bucket');
  });

  it('ttftP95Query', () => {
    expect(ttftP95Query(noFilters, '60s')).toContain('gen_ai_client_time_to_first_token_bucket');
  });

  it('tokensByModelAndTypeQuery', () => {
    expect(tokensByModelAndTypeQuery(noFilters, '3600s')).toBe(
      'sum by (gen_ai_request_model, gen_ai_token_type) (increase(gen_ai_client_token_usage_sum[3600s]))'
    );
  });
});
