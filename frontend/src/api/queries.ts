import { queryOptions } from "@tanstack/react-query";

import { api, unwrap } from "./client";

export const queryKeys = {
  me: ["me"] as const,
  users: ["users"] as const,
  userSearch: (query: string) => ["users", "search", query] as const,
  chats: ["chats"] as const,
  chat: (chatId: string) => ["chats", chatId] as const,
  messages: (chatId: string) => ["chats", chatId, "messages"] as const,
};

export const meQuery = queryOptions({
  queryKey: queryKeys.me,
  queryFn: ({ signal }) => unwrap(api.GET("/api/v1/me", { signal })),
  staleTime: Infinity,
});

export const chatsQuery = queryOptions({
  queryKey: queryKeys.chats,
  queryFn: ({ signal }) => unwrap(api.GET("/api/v1/chats", { signal })),
  refetchOnWindowFocus: true,
});

export function userSearchQuery(query: string) {
  return queryOptions({
    queryKey: queryKeys.userSearch(query),
    queryFn: ({ signal }) =>
      unwrap(api.GET("/api/v1/users/search", { params: { query: { q: query } }, signal })),
    staleTime: 30_000,
  });
}

export function chatQuery(chatId: string) {
  return queryOptions({
    queryKey: queryKeys.chat(chatId),
    queryFn: ({ signal }) =>
      unwrap(api.GET("/api/v1/chats/{chatID}", { params: { path: { chatID: chatId } }, signal })),
  });
}

export function messagesQuery(chatId: string) {
  return queryOptions({
    queryKey: queryKeys.messages(chatId),
    queryFn: ({ signal }) =>
      unwrap(
        api.GET("/api/v1/chats/{chatID}/messages", {
          params: { path: { chatID: chatId }, query: { limit: 50 } },
          signal,
        }),
      ),
  });
}
