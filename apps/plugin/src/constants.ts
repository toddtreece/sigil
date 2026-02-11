export const ROUTES = {
  Root: '',
  Conversations: 'conversations',
  Completions: 'completions',
  Traces: 'traces',
  Settings: 'settings',
} as const;

export const PAGE_TITLES = {
  [ROUTES.Conversations]: 'Conversations',
  [ROUTES.Completions]: 'Completions',
  [ROUTES.Traces]: 'Traces',
  [ROUTES.Settings]: 'Settings',
} as const;
