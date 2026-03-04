import React, { useCallback, useMemo, useState } from 'react';
import { css } from '@emotion/css';
import { dateTime, type GrafanaTheme2, type SelectableValue, type TimeRange } from '@grafana/data';
import { IconButton, Select, Stack, TimeRangePicker, useStyles2 } from '@grafana/ui';
import { type DashboardFilters, type LabelFilter } from '../../dashboard/types';
import type { DashboardDataSource } from '../../dashboard/api';
import { LabelFilterInput } from './LabelFilterInput';

export type FilterToolbarProps = {
  timeRange: TimeRange;
  filters: DashboardFilters;
  providerOptions: string[];
  modelOptions: string[];
  agentOptions: string[];
  labelKeyOptions: string[];
  labelsLoading?: boolean;
  dataSource: DashboardDataSource;
  from: number;
  to: number;
  onTimeRangeChange: (timeRange: TimeRange) => void;
  onFiltersChange: (filters: DashboardFilters) => void;
  hideLabelFilters?: boolean;
  children?: React.ReactNode;
};

export function FilterToolbar({
  timeRange,
  filters,
  providerOptions,
  modelOptions,
  agentOptions,
  labelKeyOptions,
  labelsLoading = false,
  dataSource,
  from,
  to,
  onTimeRangeChange,
  onFiltersChange,
  hideLabelFilters = false,
  children,
}: FilterToolbarProps) {
  const styles = useStyles2(getStyles);

  const [pendingRows, setPendingRows] = useState<LabelFilter[]>([]);

  const draftLabelFilters = useMemo(
    () => [...filters.labelFilters, ...pendingRows],
    [filters.labelFilters, pendingRows]
  );

  const syncLabelFilters = useCallback(
    (nextAll: LabelFilter[]) => {
      const complete: LabelFilter[] = [];
      const incomplete: LabelFilter[] = [];
      for (const lf of nextAll) {
        if (lf.key && lf.value) {
          complete.push(lf);
        } else if (lf.key || lf.value) {
          incomplete.push(lf);
        }
      }
      setPendingRows(incomplete);
      onFiltersChange({ ...filters, labelFilters: complete });
    },
    [filters, onFiltersChange]
  );

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (filters.provider) {
      count++;
    }
    if (filters.model) {
      count++;
    }
    if (filters.agentName) {
      count++;
    }
    count += filters.labelFilters.filter((lf) => lf.key && lf.value).length;
    return count;
  }, [filters]);

  const handleProviderChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, provider: value?.value ?? '', model: '', agentName: '' });
    },
    [filters, onFiltersChange]
  );
  const handleProviderCreate = useCallback(
    (value: string) => {
      onFiltersChange({ ...filters, provider: value.trim(), model: '', agentName: '' });
    },
    [filters, onFiltersChange]
  );

  const handleModelChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, model: value?.value ?? '', agentName: '' });
    },
    [filters, onFiltersChange]
  );
  const handleModelCreate = useCallback(
    (value: string) => {
      onFiltersChange({ ...filters, model: value.trim(), agentName: '' });
    },
    [filters, onFiltersChange]
  );

  const handleAgentChange = useCallback(
    (value: SelectableValue<string>) => {
      onFiltersChange({ ...filters, agentName: value?.value ?? '' });
    },
    [filters, onFiltersChange]
  );
  const handleAgentCreate = useCallback(
    (value: string) => {
      onFiltersChange({ ...filters, agentName: value.trim() });
    },
    [filters, onFiltersChange]
  );

  const handleLabelFilterChange = useCallback(
    (index: number, updated: LabelFilter) => {
      const next = [...draftLabelFilters];
      if (index >= next.length) {
        next.push(updated);
      } else {
        next[index] = updated;
      }
      syncLabelFilters(next);
    },
    [draftLabelFilters, syncLabelFilters]
  );

  const handleLabelFilterRemove = useCallback(
    (index: number) => {
      const next = draftLabelFilters.filter((_, i) => i !== index);
      syncLabelFilters(next);
    },
    [draftLabelFilters, syncLabelFilters]
  );

  const handleClearFilters = useCallback(() => {
    setPendingRows([]);
    onFiltersChange({ provider: '', model: '', agentName: '', labelFilters: [] });
  }, [onFiltersChange]);

  const providerSelectOptions = useMemo(() => providerOptions.map((v) => ({ label: v, value: v })), [providerOptions]);
  const modelSelectOptions = useMemo(() => modelOptions.map((v) => ({ label: v, value: v })), [modelOptions]);
  const agentSelectOptions = useMemo(() => agentOptions.map((v) => ({ label: v, value: v })), [agentOptions]);
  const labelKeySelectOptions = useMemo(() => labelKeyOptions.map((v) => ({ label: v, value: v })), [labelKeyOptions]);

  return (
    <div className={styles.toolbar}>
      {children && (
        <>
          {children}
          <div className={styles.divider} />
        </>
      )}
      <div className={styles.filtersSection}>
        <Stack direction="row" gap={1} alignItems="center" wrap="wrap">
          <Select<string>
            options={providerSelectOptions}
            value={filters.provider || null}
            onChange={handleProviderChange}
            onCreateOption={handleProviderCreate}
            placeholder="Provider"
            isClearable
            allowCustomValue
            isSearchable
            width={20}
          />
          <Select<string>
            options={modelSelectOptions}
            value={filters.model || null}
            onChange={handleModelChange}
            onCreateOption={handleModelCreate}
            placeholder="Model"
            isClearable
            allowCustomValue
            isSearchable
            width={20}
          />
          <Select<string>
            options={agentSelectOptions}
            value={filters.agentName || null}
            onChange={handleAgentChange}
            onCreateOption={handleAgentCreate}
            placeholder="Agent"
            isClearable
            allowCustomValue
            isSearchable
            width={20}
          />
          {!hideLabelFilters && (
            <LabelFilterInput
              filters={draftLabelFilters}
              labelKeyOptions={labelKeySelectOptions}
              labelsLoading={labelsLoading}
              dataSource={dataSource}
              from={from}
              to={to}
              onChange={handleLabelFilterChange}
              onRemove={handleLabelFilterRemove}
            />
          )}
          {activeFilterCount > 0 && (
            <IconButton
              name="times-circle"
              aria-label="Clear all filters"
              tooltip="Clear all filters"
              size="md"
              onClick={handleClearFilters}
              className={styles.clearButton}
            />
          )}
        </Stack>
      </div>
      <TimeRangePicker
        value={timeRange}
        onChange={onTimeRangeChange}
        onChangeTimeZone={() => {}}
        onMoveBackward={() => {
          const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
          const from = dateTime(timeRange.from.valueOf() - diff);
          const to = dateTime(timeRange.to.valueOf() - diff);
          onTimeRangeChange({ from, to, raw: { from, to } });
        }}
        onMoveForward={() => {
          const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
          const from = dateTime(timeRange.from.valueOf() + diff);
          const to = dateTime(timeRange.to.valueOf() + diff);
          onTimeRangeChange({ from, to, raw: { from, to } });
        }}
        onZoom={() => {
          const diff = timeRange.to.valueOf() - timeRange.from.valueOf();
          const from = dateTime(timeRange.from.valueOf() - diff / 2);
          const to = dateTime(timeRange.to.valueOf() + diff / 2);
          onTimeRangeChange({ from, to, raw: { from, to } });
        }}
        isOnCanvas
      />
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    toolbar: css({
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'space-between',
      gap: theme.spacing(1),
      padding: theme.spacing(1, 2),
      background: theme.colors.background.secondary,
      borderRadius: theme.shape.radius.default,
      border: `1px solid ${theme.colors.border.weak}`,
    }),
    filtersSection: css({
      display: 'flex',
      flex: '1 1 auto',
      minWidth: 0,
    }),
    divider: css({
      width: 1,
      alignSelf: 'stretch',
      background: theme.colors.border.medium,
    }),
    clearButton: css({
      color: theme.colors.text.secondary,
      '&:hover': {
        color: theme.colors.error.text,
      },
    }),
  };
}
