import React, { useMemo } from 'react';
import { css } from '@emotion/css';
import { type GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import type { DashboardDataSource } from '../../dashboard/api';
import type { DashboardFilters } from '../../dashboard/types';
import { type PricingMap, calculateTotalCost, calculateCostTimeSeries } from '../../dashboard/cost';
import {
  computeStep,
  computeRateInterval,
  computeRangeDuration,
  totalOpsQuery,
  totalTokensQuery,
  totalErrorsQuery,
  errorRateQuery,
  tokenUsageOverTimeQuery,
  tokenUsageByModelQuery,
  callsByProviderQuery,
  topModelsQuery,
  latencyP95Query,
  ttftP95Query,
  tokensByModelAndTypeQuery,
} from '../../dashboard/queries';
import {
  matrixToDataFrames,
  vectorToDataFrame,
  vectorToStatValue,
  statValueToDataFrame,
} from '../../dashboard/transforms';
import { usePrometheusQuery } from './usePrometheusQuery';
import { MetricPanel } from './MetricPanel';

export type DashboardGridProps = {
  dataSource: DashboardDataSource;
  filters: DashboardFilters;
  from: number;
  to: number;
  pricingMap: PricingMap;
};

const STAT_HEIGHT = 120;
const CHART_HEIGHT = 300;

export function DashboardGrid({ dataSource, filters, from, to, pricingMap }: DashboardGridProps) {
  const styles = useStyles2(getStyles);

  const step = useMemo(() => computeStep(from, to), [from, to]);
  const interval = useMemo(() => computeRateInterval(step), [step]);
  const rangeDuration = useMemo(() => computeRangeDuration(from, to), [from, to]);

  // Stat queries (instant)
  const totalOps = usePrometheusQuery(dataSource, totalOpsQuery(filters, rangeDuration), from, to, 'instant');
  const totalTokens = usePrometheusQuery(dataSource, totalTokensQuery(filters, rangeDuration), from, to, 'instant');
  const totalErrors = usePrometheusQuery(dataSource, totalErrorsQuery(filters, rangeDuration), from, to, 'instant');
  const errRate = usePrometheusQuery(dataSource, errorRateQuery(filters, rangeDuration), from, to, 'instant');

  // Cost query (instant, per model+type for client-side pricing)
  const costTokens = usePrometheusQuery(
    dataSource,
    tokensByModelAndTypeQuery(filters, rangeDuration),
    from,
    to,
    'instant'
  );

  // Timeseries queries (range)
  const tokensOverTime = usePrometheusQuery(
    dataSource,
    tokenUsageOverTimeQuery(filters, interval),
    from,
    to,
    'range',
    step
  );
  const tokensByModel = usePrometheusQuery(
    dataSource,
    tokenUsageByModelQuery(filters, interval),
    from,
    to,
    'range',
    step
  );
  const latency = usePrometheusQuery(dataSource, latencyP95Query(filters, interval), from, to, 'range', step);
  const ttft = usePrometheusQuery(dataSource, ttftP95Query(filters, interval), from, to, 'range', step);

  // Piechart queries (instant)
  const byProvider = usePrometheusQuery(dataSource, callsByProviderQuery(filters, rangeDuration), from, to, 'instant');
  const byModel = usePrometheusQuery(dataSource, topModelsQuery(filters, rangeDuration), from, to, 'instant');

  // Computed cost
  const totalCostValue = useMemo(() => {
    if (!costTokens.data) {
      return 0;
    }
    return calculateTotalCost(costTokens.data, pricingMap);
  }, [costTokens.data, pricingMap]);

  const costTimeSeries = useMemo(() => {
    if (!tokensByModel.data) {
      return [];
    }
    return [calculateCostTimeSeries(tokensByModel.data, pricingMap)];
  }, [tokensByModel.data, pricingMap]);

  return (
    <div className={styles.grid}>
      {/* Stat row */}
      <div className={styles.statRow}>
        <MetricPanel
          title="Total Operations"
          pluginId="stat"
          height={STAT_HEIGHT}
          loading={totalOps.loading}
          error={totalOps.error}
          data={[statValueToDataFrame(totalOps.data ? vectorToStatValue(totalOps.data) : 0, 'Operations')]}
        />
        <MetricPanel
          title="Total Tokens"
          pluginId="stat"
          height={STAT_HEIGHT}
          loading={totalTokens.loading}
          error={totalTokens.error}
          data={[statValueToDataFrame(totalTokens.data ? vectorToStatValue(totalTokens.data) : 0, 'Tokens')]}
        />
        <MetricPanel
          title="Total Errors"
          pluginId="stat"
          height={STAT_HEIGHT}
          loading={totalErrors.loading}
          error={totalErrors.error}
          data={[statValueToDataFrame(totalErrors.data ? vectorToStatValue(totalErrors.data) : 0, 'Errors')]}
          fieldConfig={{
            defaults: { color: { mode: 'fixed', fixedColor: 'red' } },
            overrides: [],
          }}
        />
        <MetricPanel
          title="Error Rate"
          pluginId="stat"
          height={STAT_HEIGHT}
          loading={errRate.loading}
          error={errRate.error}
          data={[statValueToDataFrame(errRate.data ? vectorToStatValue(errRate.data) : 0, 'Error Rate', 'percent')]}
        />
        <MetricPanel
          title="Estimated Cost"
          pluginId="stat"
          height={STAT_HEIGHT}
          loading={costTokens.loading}
          error={costTokens.error}
          data={[statValueToDataFrame(totalCostValue, 'Cost', 'currencyUSD')]}
        />
      </div>

      {/* Token usage over time */}
      <MetricPanel
        title="Token Usage Over Time"
        description="Breakdown by token type (input, output, cache, reasoning)"
        pluginId="timeseries"
        height={CHART_HEIGHT}
        loading={tokensOverTime.loading}
        error={tokensOverTime.error}
        data={tokensOverTime.data ? matrixToDataFrames(tokensOverTime.data) : []}
      />

      {/* Cost over time */}
      <MetricPanel
        title="Estimated Cost Over Time"
        description="Cost computed from token rates and model card pricing"
        pluginId="timeseries"
        height={CHART_HEIGHT}
        loading={tokensByModel.loading}
        error={tokensByModel.error}
        data={costTimeSeries}
      />

      {/* Two-column row: piecharts */}
      <div className={styles.twoColRow}>
        <MetricPanel
          title="Calls by Provider"
          pluginId="piechart"
          height={CHART_HEIGHT}
          loading={byProvider.loading}
          error={byProvider.error}
          data={byProvider.data ? [vectorToDataFrame(byProvider.data, 'Calls')] : []}
        />
        <MetricPanel
          title="Top Models"
          pluginId="piechart"
          height={CHART_HEIGHT}
          loading={byModel.loading}
          error={byModel.error}
          data={byModel.data ? [vectorToDataFrame(byModel.data, 'Calls')] : []}
        />
      </div>

      {/* Two-column row: latency */}
      <div className={styles.twoColRow}>
        <MetricPanel
          title="Latency P95"
          description="95th percentile operation duration"
          pluginId="timeseries"
          height={CHART_HEIGHT}
          loading={latency.loading}
          error={latency.error}
          data={latency.data ? matrixToDataFrames(latency.data) : []}
          fieldConfig={{
            defaults: { unit: 's' },
            overrides: [],
          }}
        />
        <MetricPanel
          title="Time to First Token P95"
          description="95th percentile TTFT (streaming only)"
          pluginId="timeseries"
          height={CHART_HEIGHT}
          loading={ttft.loading}
          error={ttft.error}
          data={ttft.data ? matrixToDataFrames(ttft.data) : []}
          fieldConfig={{
            defaults: { unit: 's' },
            overrides: [],
          }}
        />
      </div>
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    grid: css({
      display: 'flex',
      flexDirection: 'column',
      gap: theme.spacing(1),
    }),
    statRow: css({
      display: 'grid',
      gridTemplateColumns: 'repeat(5, 1fr)',
      gap: theme.spacing(1),
    }),
    twoColRow: css({
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: theme.spacing(1),
    }),
  };
}
