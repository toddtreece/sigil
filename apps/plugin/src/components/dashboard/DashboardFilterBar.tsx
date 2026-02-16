import React, { useCallback, useMemo } from 'react';
import { TimeRangePicker, Select, Stack } from '@grafana/ui';
import { dateTimeParse, type SelectableValue, type TimeRange } from '@grafana/data';
import type { DashboardFilters } from '../../dashboard/types';

export type DashboardFilterBarProps = {
  timeRange: TimeRange;
  filters: DashboardFilters;
  providerOptions: string[];
  modelOptions: string[];
  agentOptions: string[];
  onTimeRangeChange: (timeRange: TimeRange) => void;
  onFiltersChange: (filters: DashboardFilters) => void;
};

export function DashboardFilterBar({
  timeRange,
  filters,
  providerOptions,
  modelOptions,
  agentOptions,
  onTimeRangeChange,
  onFiltersChange,
}: DashboardFilterBarProps) {
  const handleProviderChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, provider: value?.value ?? '' });
    },
    [filters, onFiltersChange]
  );

  const handleModelChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, model: value?.value ?? '' });
    },
    [filters, onFiltersChange]
  );

  const handleAgentChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, agentName: value?.value ?? '' });
    },
    [filters, onFiltersChange]
  );

  const providerSelectOptions = useMemo(() => providerOptions.map((v) => ({ label: v, value: v })), [providerOptions]);
  const modelSelectOptions = useMemo(() => modelOptions.map((v) => ({ label: v, value: v })), [modelOptions]);
  const agentSelectOptions = useMemo(() => agentOptions.map((v) => ({ label: v, value: v })), [agentOptions]);

  const onMoveBackward = useCallback(() => {
    const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
    onTimeRangeChange({
      from: dateTimeParse(timeRange.from.valueOf() - diff),
      to: dateTimeParse(timeRange.to.valueOf() - diff),
      raw: {
        from: dateTimeParse(timeRange.from.valueOf() - diff),
        to: dateTimeParse(timeRange.to.valueOf() - diff),
      },
    });
  }, [timeRange, onTimeRangeChange]);

  const onMoveForward = useCallback(() => {
    const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
    onTimeRangeChange({
      from: dateTimeParse(timeRange.from.valueOf() + diff),
      to: dateTimeParse(timeRange.to.valueOf() + diff),
      raw: {
        from: dateTimeParse(timeRange.from.valueOf() + diff),
        to: dateTimeParse(timeRange.to.valueOf() + diff),
      },
    });
  }, [timeRange, onTimeRangeChange]);

  const onZoom = useCallback(() => {
    const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
    const center = timeRange.from.valueOf() + diff / 2;
    onTimeRangeChange({
      from: dateTimeParse(center - diff),
      to: dateTimeParse(center + diff),
      raw: {
        from: dateTimeParse(center - diff),
        to: dateTimeParse(center + diff),
      },
    });
  }, [timeRange, onTimeRangeChange]);

  return (
    <Stack direction="row" gap={1} alignItems="center" wrap="wrap">
      <TimeRangePicker
        value={timeRange}
        onChange={onTimeRangeChange}
        onChangeTimeZone={() => {}}
        onMoveBackward={onMoveBackward}
        onMoveForward={onMoveForward}
        onZoom={onZoom}
      />
      <Select<string>
        options={providerSelectOptions}
        value={filters.provider || null}
        onChange={handleProviderChange}
        placeholder="Provider"
        isClearable
        width={20}
      />
      <Select<string>
        options={modelSelectOptions}
        value={filters.model || null}
        onChange={handleModelChange}
        placeholder="Model"
        isClearable
        width={24}
      />
      <Select<string>
        options={agentSelectOptions}
        value={filters.agentName || null}
        onChange={handleAgentChange}
        placeholder="Agent"
        isClearable
        width={20}
      />
    </Stack>
  );
}
