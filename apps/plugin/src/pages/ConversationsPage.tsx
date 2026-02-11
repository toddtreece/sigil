import React from 'react';
import { Alert, Stack, Text } from '@grafana/ui';

export default function ConversationsPage() {
  return (
    <Stack direction="column" gap={2}>
      <h2>Conversations</h2>
      <Alert severity="info" title="Sigil bootstrap placeholder">
        <Text>
          Conversation browsing is scaffolded. This page will display LLM conversation summaries stored by Sigil.
        </Text>
      </Alert>
    </Stack>
  );
}
