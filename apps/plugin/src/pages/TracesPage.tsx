import React from 'react';
import { Alert, Stack, Text } from '@grafana/ui';

export default function TracesPage() {
  return (
    <Stack direction="column" gap={2}>
      <h2>Traces</h2>
      <Alert severity="info" title="Sigil bootstrap placeholder">
        <Text>
          Trace exploration is scaffolded. This page will correlate OTLP traces with externalized Records payloads.
        </Text>
      </Alert>
    </Stack>
  );
}
