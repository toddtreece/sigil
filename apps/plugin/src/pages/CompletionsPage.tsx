import React from 'react';
import { Alert, Stack, Text } from '@grafana/ui';

export default function CompletionsPage() {
  return (
    <Stack direction="column" gap={2}>
      <h2>Completions</h2>
      <Alert severity="info" title="Sigil bootstrap placeholder">
        <Text>
          Completion inspection is scaffolded. This page will show action messages and completion payload references.
        </Text>
      </Alert>
    </Stack>
  );
}
