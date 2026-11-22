import type { Chat, User } from "../api/types";

export const me: User = {
  id: "11111111-1111-4111-8111-111111111111",
  username: "sam",
  created_at: "2026-11-01T09:00:00Z",
};

export const alice = { id: "22222222-2222-4222-8222-222222222222", username: "alice" };
export const bob = { id: "33333333-3333-4333-8333-333333333333", username: "bob" };

export function makeChat(overrides: Partial<Chat> & Pick<Chat, "id" | "type">): Chat {
  return {
    title: null,
    created_by: me.id,
    created_at: "2026-11-01T09:00:00Z",
    unread_count: 0,
    muted: false,
    mute_version: 0,
    ...overrides,
  };
}
