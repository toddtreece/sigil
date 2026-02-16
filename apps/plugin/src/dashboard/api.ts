import { lastValueFrom } from 'rxjs';
import { getBackendSrv } from '@grafana/runtime';
import type { ModelCard, ModelCardListResponse, PrometheusLabelValuesResponse, PrometheusQueryResponse } from './types';

const queryBasePath = '/api/plugins/grafana-sigil-app/resources/query';

export type DashboardDataSource = {
  queryRange: (query: string, start: number, end: number, step: number) => Promise<PrometheusQueryResponse>;
  queryInstant: (query: string, time: number) => Promise<PrometheusQueryResponse>;
  labelValues: (label: string, start: number, end: number) => Promise<string[]>;
  listModelCards: () => Promise<ModelCard[]>;
};

export const defaultDashboardDataSource: DashboardDataSource = {
  async queryRange(query, start, end, step) {
    const response = await lastValueFrom(
      getBackendSrv().fetch<PrometheusQueryResponse>({
        method: 'GET',
        url: `${queryBasePath}/proxy/prometheus/api/v1/query_range`,
        params: { query, start, end, step },
      })
    );
    return response.data;
  },

  async queryInstant(query, time) {
    const response = await lastValueFrom(
      getBackendSrv().fetch<PrometheusQueryResponse>({
        method: 'GET',
        url: `${queryBasePath}/proxy/prometheus/api/v1/query`,
        params: { query, time },
      })
    );
    return response.data;
  },

  async labelValues(label, start, end) {
    const response = await lastValueFrom(
      getBackendSrv().fetch<PrometheusLabelValuesResponse>({
        method: 'GET',
        url: `${queryBasePath}/proxy/prometheus/api/v1/label/${encodeURIComponent(label)}/values`,
        params: { start, end },
      })
    );
    return response.data.data ?? [];
  },

  async listModelCards() {
    const allCards: ModelCard[] = [];
    let cursor = '';
    do {
      const params: Record<string, string | number> = { limit: 200 };
      if (cursor) {
        params.cursor = cursor;
      }
      const response = await lastValueFrom(
        getBackendSrv().fetch<ModelCardListResponse>({
          method: 'GET',
          url: `${queryBasePath}/model-cards`,
          params,
        })
      );
      allCards.push(...(response.data.data ?? []));
      cursor = response.data.next_cursor ?? '';
    } while (cursor);
    return allCards;
  },
};
