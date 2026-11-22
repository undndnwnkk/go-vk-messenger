import type { Chat } from "../api/types";

export function chatTitle(chat: Chat): string {
  return chat.title ?? "Direct chat";
}

export function chatSubtitle(chat: Chat): string {
  return chat.type === "group" ? "Group" : "Direct message";
}
