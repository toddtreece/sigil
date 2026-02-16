import { buildPricingMap, lookupPricing, calculateTotalCost, calculateCostTimeSeries } from './cost';
import type { ModelCard, ModelCardPricing, PrometheusQueryResponse } from './types';

function makeCard(sourceModelId: string, provider: string, pricing: Partial<ModelCardPricing>): ModelCard {
  return {
    model_key: `openrouter:${sourceModelId}`,
    source: 'openrouter',
    source_model_id: sourceModelId,
    canonical_slug: sourceModelId.replace('/', '-'),
    name: sourceModelId.split('/').pop() ?? sourceModelId,
    provider,
    pricing: {
      prompt_usd_per_token: null,
      completion_usd_per_token: null,
      request_usd: null,
      image_usd: null,
      web_search_usd: null,
      input_cache_read_usd_per_token: null,
      input_cache_write_usd_per_token: null,
      ...pricing,
    },
    is_free: false,
  };
}

describe('buildPricingMap', () => {
  it('indexes by source_model_id and model part', () => {
    const cards = [makeCard('openai/gpt-4o', 'openai', { prompt_usd_per_token: 0.0025 })];
    const map = buildPricingMap(cards);

    expect(map.get('openai/gpt-4o')).toBeDefined();
    expect(map.get('gpt-4o')).toBeDefined();
    expect(map.get('openai-gpt-4o')).toBeDefined(); // canonical_slug
  });
});

describe('lookupPricing', () => {
  const cards = [makeCard('openai/gpt-4o', 'openai', { prompt_usd_per_token: 0.0025 })];
  const map = buildPricingMap(cards);

  it('finds by direct model name', () => {
    expect(lookupPricing(map, 'gpt-4o')).toBeDefined();
  });

  it('finds by provider/model format', () => {
    expect(lookupPricing(map, 'gpt-4o', 'openai')).toBeDefined();
  });

  it('returns undefined for unknown model', () => {
    expect(lookupPricing(map, 'unknown-model')).toBeUndefined();
  });
});

describe('calculateTotalCost', () => {
  const cards = [
    makeCard('openai/gpt-4o', 'openai', {
      prompt_usd_per_token: 0.0025,
      completion_usd_per_token: 0.01,
      input_cache_read_usd_per_token: 0.00125,
      input_cache_write_usd_per_token: 0.00375,
    }),
  ];
  const pricingMap = buildPricingMap(cards);

  it('computes cost from token counts', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: {
        resultType: 'vector',
        result: [
          { metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'input' }, value: [0, '1000'] },
          { metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'output' }, value: [0, '500'] },
        ],
      },
    };

    const cost = calculateTotalCost(response, pricingMap);
    // 1000 * 0.0025 + 500 * 0.01 = 2.5 + 5 = 7.5
    expect(cost).toBeCloseTo(7.5);
  });

  it('skips unknown models', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: {
        resultType: 'vector',
        result: [{ metric: { gen_ai_request_model: 'unknown-model', gen_ai_token_type: 'input' }, value: [0, '1000'] }],
      },
    };

    expect(calculateTotalCost(response, pricingMap)).toBe(0);
  });

  it('handles cache token types', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: {
        resultType: 'vector',
        result: [
          { metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'cache_read' }, value: [0, '2000'] },
          { metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'cache_write' }, value: [0, '1000'] },
        ],
      },
    };

    const cost = calculateTotalCost(response, pricingMap);
    // 2000 * 0.00125 + 1000 * 0.00375 = 2.5 + 3.75 = 6.25
    expect(cost).toBeCloseTo(6.25);
  });

  it('returns 0 for empty response', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: { resultType: 'vector', result: [] },
    };
    expect(calculateTotalCost(response, pricingMap)).toBe(0);
  });
});

describe('calculateCostTimeSeries', () => {
  const cards = [
    makeCard('openai/gpt-4o', 'openai', {
      prompt_usd_per_token: 0.0025,
      completion_usd_per_token: 0.01,
    }),
  ];
  const pricingMap = buildPricingMap(cards);

  it('aggregates cost across series into a single timeseries', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: {
        resultType: 'matrix',
        result: [
          {
            metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'input' },
            values: [
              [1000, '100'],
              [1060, '200'],
            ],
          },
          {
            metric: { gen_ai_request_model: 'gpt-4o', gen_ai_token_type: 'output' },
            values: [
              [1000, '50'],
              [1060, '75'],
            ],
          },
        ],
      },
    };

    const frame = calculateCostTimeSeries(response, pricingMap);
    expect(frame.fields).toHaveLength(2);

    const times = frame.fields[0].values;
    const costs = frame.fields[1].values;

    expect(times).toEqual([1000000, 1060000]);
    // t=1000: 100*0.0025 + 50*0.01 = 0.25 + 0.5 = 0.75
    // t=1060: 200*0.0025 + 75*0.01 = 0.5 + 0.75 = 1.25
    expect(costs[0]).toBeCloseTo(0.75);
    expect(costs[1]).toBeCloseTo(1.25);
  });

  it('returns empty frame for non-matrix response', () => {
    const response: PrometheusQueryResponse = {
      status: 'success',
      data: { resultType: 'vector', result: [] },
    };
    const frame = calculateCostTimeSeries(response, pricingMap);
    expect(frame.fields).toHaveLength(0);
  });
});
