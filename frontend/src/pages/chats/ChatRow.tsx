import { NavLink } from "react-router";

import type { Chat } from "../../api/types";
import { Avatar } from "../../components/Avatar";
import { Icon } from "../../components/Icon";
import { chatSubtitle, chatTitle } from "../../lib/chat";

export function ChatRow({ chat }: { chat: Chat }) {
  const title = chatTitle(chat);
  const unread = chat.unread_count > 0;
  return (
    <NavLink
      to={`/chats/${chat.id}`}
      className={({ isActive }) =>
        `flex h-row w-full flex-none items-center gap-3 px-4 text-left no-underline ${isActive ? "bg-row-active" : "hover:bg-row-hover"}`
      }
    >
      <Avatar seed={chat.id} name={title} shape={chat.type === "group" ? "square" : "round"} />
      <div className="min-w-0 flex-1">
        <div
          className={`truncate text-md leading-5 text-text ${unread ? "font-semibold" : "font-medium"}`}
        >
          {title}
        </div>
        <div className="mt-0.5 flex items-center gap-2">
          <span
            className={`min-w-0 flex-1 truncate text-base ${unread ? "text-text-2" : "text-text-3"}`}
          >
            {chatSubtitle(chat)}
          </span>
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
