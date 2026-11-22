import { NavLink } from "react-router";

import type { Chat } from "../../api/types";
import { Avatar } from "../../components/Avatar";
import { Icon } from "../../components/Icon";
import { chatAvatarSeed, chatPreview, chatTitle, lastActivity } from "../../lib/chat";
import { formatListTime } from "../../lib/time";

interface ChatRowProps {
  chat: Chat;
  currentUserId: string | undefined;
}

function Preview({ chat, currentUserId }: ChatRowProps) {
  const preview = chatPreview(chat, currentUserId);
  const unread = chat.unread_count > 0;
  const tone = unread ? "text-text-2" : "text-text-3";
  if (preview.kind === "empty") {
    return <span className="min-w-0 flex-1 truncate text-base text-text-3">No messages yet</span>;
  }
  if (preview.kind === "deleted") {
    return (
      <span className={`min-w-0 flex-1 truncate text-base italic ${tone}`}>Message deleted</span>
    );
  }
  return (
    <span className={`min-w-0 flex-1 truncate text-base ${tone}`}>
      {preview.prefix !== null && `${preview.prefix}: `}
      {preview.text}
    </span>
  );
}

export function ChatRow({ chat, currentUserId }: ChatRowProps) {
  const title = chatTitle(chat);
  const unread = chat.unread_count > 0;
  const activity = lastActivity(chat);
  const highlightTime = unread && !chat.muted;
  return (
    <NavLink
      to={`/chats/${chat.id}`}
      className={({ isActive }) =>
        `flex h-row w-full flex-none items-center gap-3 px-4 text-left no-underline ${isActive ? "bg-row-active" : "hover:bg-row-hover"}`
      }
    >
      <Avatar
        seed={chatAvatarSeed(chat)}
        name={title}
        shape={chat.type === "group" ? "square" : "round"}
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span
            className={`min-w-0 flex-1 truncate text-md leading-5 text-text ${unread ? "font-semibold" : "font-medium"}`}
          >
            {title}
          </span>
          <time
            dateTime={activity.toISOString()}
            className={`flex-none text-xs ${highlightTime ? "font-semibold text-accent" : "text-text-3"}`}
          >
            {formatListTime(activity)}
          </time>
        </div>
        <div className="mt-0.5 flex items-center gap-2">
          <Preview chat={chat} currentUserId={currentUserId} />
          {chat.muted && (
            <span className="text-text-3">
              <Icon name="muted" size={14} />
              <span className="sr-only">Muted</span>
            </span>
          )}
          {unread && (
            <span
              className={`h-5 min-w-5 flex-none rounded-[10px] px-1.5 text-center text-xs leading-5 font-semibold ${chat.muted ? "bg-badge-muted-bg text-badge-muted-fg" : "bg-badge-bg text-badge-fg"}`}
            >
              <span className="sr-only">Unread messages: </span>
              {chat.unread_count}
            </span>
          )}
        </div>
      </div>
    </NavLink>
  );
}
