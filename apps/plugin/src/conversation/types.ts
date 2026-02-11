export type ConversationMessage = {
  id: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  timestamp: string;
};
