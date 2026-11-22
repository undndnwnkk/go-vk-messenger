import { useQuery } from "@tanstack/react-query";

import { chatsQuery } from "../../api/queries";
import { useAuth } from "../../auth/useAuth";
import { Button } from "../../components/Button";
import { ErrorState } from "../../components/ErrorState";
import { Icon } from "../../components/Icon";
import { StatePanel } from "../../components/StatePanel";
import { chatTitle } from "../../lib/chat";
import { ChatRow } from "./ChatRow";
import type { NewChatMode } from "./new/NewChatDialog";

function EmptyIllustration() {
  return (
    <svg
      width="120"
      height="96"
      viewBox="0 0 120 96"
      aria-hidden="true"
      className="block flex-none"
    >
      <rect x="8" y="8" width="72" height="46" rx="16" fill="var(--sunken)" />
      <path d="M22 52L18 66L34 54Z" fill="var(--sunken)" />
      <rect x="40" y="34" width="72" height="46" rx="16" fill="var(--accent-soft)" />
      <path d="M98 78L104 90L88 80Z" fill="var(--accent-soft)" />
      <rect x="54" y="49" width="42" height="6" rx="3" fill="var(--accent)" opacity=".6" />
      <rect x="54" y="61" width="26" height="6" rx="3" fill="var(--accent)" opacity=".35" />
    </svg>
  );
}

function SkeletonRows() {
  return Array.from({ length: 8 }, (_, index) => (
    <div key={index} className="flex h-row flex-none items-center gap-3 px-4">
      <div className="size-avatar-lg flex-none animate-pulse rounded-full bg-skeleton" />
      <div className="flex flex-1 flex-col gap-2">
        <div className="h-3 w-2/5 animate-pulse rounded-pill bg-skeleton" />
        <div className="h-3 w-4/5 animate-pulse rounded-pill bg-skeleton" />
      </div>
    </div>
  ));
}

interface ChatListProps {
  filter: string;
  onNewChat: (mode: NewChatMode) => void;
}

export function ChatList({ filter, onNewChat }: ChatListProps) {
  const query = useQuery(chatsQuery);
  const { user } = useAuth();
  const needle = filter.trim().toLowerCase();

  let content;
  if (query.isPending) {
    content = <SkeletonRows />;
  } else if (query.isError) {
    content = (
      <ErrorState
        title="Couldn't load chats"
        description="Something went wrong on our side. Check your connection and try again."
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  } else if (query.data.length === 0) {
    content = (
      <StatePanel
        visual={<EmptyIllustration />}
        title="No chats yet"
        description="Conversations you start, or are invited to, will show up here."
      >
        <Button
          icon={<Icon name="compose" size={18} />}
          onClick={() => {
            onNewChat("direct");
          }}
        >
          New chat
        </Button>
        <Button
          variant="secondary"
          icon={<Icon name="group" size={18} />}
          onClick={() => {
            onNewChat("group");
          }}
        >
          Create group
        </Button>
      </StatePanel>
    );
  } else {
    const chats = needle
      ? query.data.filter((chat) => chatTitle(chat).toLowerCase().includes(needle))
      : query.data;
    content =
      chats.length === 0 ? (
        <p className="px-5 py-6 text-center text-base text-text-3">
          No chats match “{filter.trim()}”.
        </p>
      ) : (
        chats.map((chat) => <ChatRow key={chat.id} chat={chat} currentUserId={user?.id} />)
      );
  }

  return (
    <nav
      aria-label="Chats"
      aria-busy={query.isPending}
      className="flex min-h-0 flex-1 [scrollbar-width:thin] flex-col overflow-y-auto"
    >
      {content}
    </nav>
  );
}
