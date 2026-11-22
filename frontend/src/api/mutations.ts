import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "./client";
import { queryKeys } from "./queries";

function useInvalidateAfterChatCreated() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.chats, exact: true }),
      queryClient.invalidateQueries({ queryKey: queryKeys.users }),
    ]);
}

export function useCreateDirectChat() {
  const invalidate = useInvalidateAfterChatCreated();
  return useMutation({
    mutationFn: (userId: string) =>
      unwrap(api.POST("/api/v1/chats/direct", { body: { user2_id: userId } })),
    onSuccess: invalidate,
  });
}

export function useCreateGroupChat() {
  const invalidate = useInvalidateAfterChatCreated();
  return useMutation({
    mutationFn: (input: { title: string; userIds: string[] }) =>
      unwrap(
        api.POST("/api/v1/chats/group", { body: { title: input.title, user_ids: input.userIds } }),
      ),
    onSuccess: invalidate,
  });
}

export function useMarkChatRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { chatId: string; messageId: number }) =>
      unwrap(
        api.POST("/api/v1/chats/{chatID}/read", {
          params: { path: { chatID: input.chatId } },
          body: { message_id: input.messageId },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.chats, exact: true }),
  });
}
