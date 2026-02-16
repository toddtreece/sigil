import { MutableDataFrame, FieldType, type DataFrame } from '@grafana/data';
import type { PrometheusMatrixResult, PrometheusQueryResponse, PrometheusVectorResult } from './types';

/** Build a display label from a set of Prometheus labels. */
function labelString(metric: Record<string, string>): string {
  // Use the first non-empty label value, or join all.
  const vals = Object.values(metric).filter(Boolean);
  return vals.join(' / ') || 'Value';
}

/** Convert a Prometheus matrix (query_range) response to Grafana DataFrames. */
export function matrixToDataFrames(response: PrometheusQueryResponse): DataFrame[] {
  if (response.data.resultType !== 'matrix') {
    return [];
  }
  const results = response.data.result as PrometheusMatrixResult[];

  return results.map((series) => {
    const name = labelString(series.metric);
    const times: number[] = [];
    const values: number[] = [];

    for (const [ts, val] of series.values) {
      times.push(ts * 1000); // Prometheus uses seconds, Grafana uses milliseconds
      values.push(parseFloat(val));
    }

    return new MutableDataFrame({
      name,
      fields: [
        { name: 'Time', type: FieldType.time, values: times },
        { name: name, type: FieldType.number, values, labels: series.metric },
      ],
    });
  });
}

/** Convert a Prometheus vector (instant) response to a single DataFrame for pie/bar charts. */
export function vectorToDataFrame(response: PrometheusQueryResponse, valueName = 'Value'): DataFrame {
  if (response.data.resultType !== 'vector') {
    return new MutableDataFrame({ fields: [] });
  }
  const results = response.data.result as PrometheusVectorResult[];

  const labels: string[] = [];
  const values: number[] = [];

  for (const result of results) {
    labels.push(labelString(result.metric));
    values.push(parseFloat(result.value[1]));
  }

  return new MutableDataFrame({
    fields: [
      { name: 'Label', type: FieldType.string, values: labels },
      { name: valueName, type: FieldType.number, values },
    ],
  });
}

/** Extract a single scalar from a Prometheus vector response. */
export function vectorToStatValue(response: PrometheusQueryResponse): number {
  if (response.data.resultType !== 'vector') {
    return 0;
  }
  const results = response.data.result as PrometheusVectorResult[];
  if (results.length === 0) {
    return 0;
  }
  return parseFloat(results[0].value[1]);
}

/** Convert a single scalar value to a stat-compatible DataFrame. */
export function statValueToDataFrame(value: number, name: string, unit?: string): DataFrame {
  return new MutableDataFrame({
    fields: [
      {
        name,
        type: FieldType.number,
        values: [value],
        config: unit ? { unit } : {},
      },
    ],
  });
}
