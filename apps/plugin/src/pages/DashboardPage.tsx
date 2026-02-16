import React, { useCallback, useMemo, useState } from 'react';
import { css } from '@emotion/css';
import { dateTimeParse, type GrafanaTheme2, type TimeRange } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import { type DashboardDataSource, defaultDashboardDataSource } from '../dashboard/api';
import { type DashboardFilters, emptyFilters } from '../dashboard/types';
import { DashboardFilterBar } from '../components/dashboard/DashboardFilterBar';
import { DashboardGrid } from '../components/dashboard/DashboardGrid';
import { useModelCards } from '../components/dashboard/useModelCards';
import { useLabelValues } from '../components/dashboard/useLabelValues';

type DashboardPageProps = {
  dataSource?: DashboardDataSource;
};

const defaultTimeRange: TimeRange = {
  from: dateTimeParse('now-1h'),
  to: dateTimeParse('now'),
  raw: { from: 'now-1h', to: 'now' },
};

export default function DashboardPage({ dataSource = defaultDashboardDataSource }: DashboardPageProps) {
  const styles = useStyles2(getStyles);
  const [timeRange, setTimeRange] = useState<TimeRange>(defaultTimeRange);
  const [filters, setFilters] = useState<DashboardFilters>(emptyFilters);

  const from = useMemo(() => Math.floor(timeRange.from.valueOf() / 1000), [timeRange]);
  const to = useMemo(() => Math.floor(timeRange.to.valueOf() / 1000), [timeRange]);

  const { pricingMap } = useModelCards(dataSource);

  const providerValues = useLabelValues(dataSource, 'gen_ai_provider_name', from, to);
  const modelValues = useLabelValues(dataSource, 'gen_ai_request_model', from, to);
  const agentValues = useLabelValues(dataSource, 'gen_ai_agent_name', from, to);

  const handleTimeRangeChange = useCallback((newTimeRange: TimeRange) => {
    setTimeRange(newTimeRange);
  }, []);

  const handleFiltersChange = useCallback((newFilters: DashboardFilters) => {
    setFilters(newFilters);
  }, []);

  return (
    <div className={styles.container}>
      <DashboardFilterBar
        timeRange={timeRange}
        filters={filters}
        providerOptions={providerValues.values}
        modelOptions={modelValues.values}
        agentOptions={agentValues.values}
        onTimeRangeChange={handleTimeRangeChange}
        onFiltersChange={handleFiltersChange}
      />
      <DashboardGrid dataSource={dataSource} filters={filters} from={from} to={to} pricingMap={pricingMap} />
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    container: css({
      padding: theme.spacing(2),
    }),
  };
}
