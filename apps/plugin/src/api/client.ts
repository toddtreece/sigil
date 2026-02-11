export type SigilConversation = {
  id: string;
  title: string;
  updatedAt: string;
};

export type SigilCompletion = {
  id: string;
  conversationId: string;
  model: string;
  createdAt: string;
};
