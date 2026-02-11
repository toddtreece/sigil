import type { AppPlugin, KeyValue } from '@grafana/data';

export async function bootstrap<T extends KeyValue>(_plugin: AppPlugin<T>): Promise<void> {
  // Reserved for extension registration and runtime wiring.
}
