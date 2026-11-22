import type { Chat } from "../api/types";

export function chatTitle(chat: Chat): string {
  if (chat.type === "group") {
    return chat.title ?? "Group";
  }
  return chat.peer?.username ?? "Direct chat";
}

export function chatSubtitle(chat: Chat): string {
  if (chat.type === "group") {
    return "Group";
  }
  return chat.peer ? `@${chat.peer.username}` : "Direct message";
}

export function chatAvatarSeed(chat: Chat): string {
  return chat.type === "direct" && chat.peer ? chat.peer.id : chat.id;
}

export function lastActivity(chat: Chat): Date {
  return new Date(chat.last_message?.created_at ?? chat.created_at);
}

export type ChatPreview =
  { kind: "empty" } | { kind: "deleted" } | { kind: "text"; prefix: string | null; text: string };

export function chatPreview(chat: Chat, currentUserId: string | undefined): ChatPreview {
  const message = chat.last_message;
  if (message === undefined) {
    return { kind: "empty" };
  }
  if (message.deleted) {
    return { kind: "deleted" };
  }
  let prefix: string | null = null;
  if (message.sender_id === currentUserId) {
    prefix = "You";
  } else if (chat.type === "group") {
    prefix = message.sender_username;
  }
  return { kind: "text", prefix, text: message.content.replace(/\s+/g, " ").trim() };
}
