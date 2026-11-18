import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router";

import { isApiError } from "../../api/errors";
import { chatQuery, messagesQuery } from "../../api/queries";
import { useAuth } from "../../auth/useAuth";
import { Avatar } from "../../components/Avatar";
import { ErrorState } from "../../components/ErrorState";
import { Icon } from "../../components/Icon";
import { StatePanel } from "../../components/StatePanel";
import { chatSubtitle, chatTitle } from "../../lib/chat";
import { MessageList } from "./MessageList";

function isMissingChat(error: unknown): boolean {
  return isApiError(error) && (error.status === 404 || error.code === "invalid_id");
}

function MessageSkeleton() {
  const widths = ["w-2/5", "w-1/2", "w-1/3", "w-3/5", "w-1/4"];
  return (
    <div aria-hidden="true" className="flex flex-col gap-3">
      {widths.map((width, index) => (
        <div key={width} className={`flex ${index % 2 === 0 ? "justify-start" : "justify-end"}`}>
          <div className={`h-10 animate-pulse rounded-lg bg-skeleton ${width}`} />
        </div>
      ))}
    </div>
  );
}

export function ChatPage() {
  const { id = "" } = useParams();
  const { user } = useAuth();
  const chat = useQuery(chatQuery(id));
  const messages = useQuery(messagesQuery(id));

  if (chat.isError) {
    if (isMissingChat(chat.error)) {
      return (
        <StatePanel
          visual={null}
          title="Chat not found"
          description="This chat doesn't exist or you're not a member."
        >
          <Link
            to="/chats"
            className="inline-flex h-touch items-center justify-center rounded-md border border-border-input bg-surface font-semibold text-text no-underline"
          >
            Back to chats
          </Link>
        </StatePanel>
      );
    }
    return (
      <ErrorState
        title="Chat didn't load"
        description="This conversation could not be opened right now."
        error={chat.error}
        onRetry={() => void chat.refetch()}
        retryLabel="Retry"
      />
    );
  }

  const title = chat.data ? chatTitle(chat.data) : "";

  let body;
  if (messages.isPending) {
    body = <MessageSkeleton />;
  } else if (messages.isError) {
    body = (
      <ErrorState
        title="Messages didn't load"
        description="This conversation could not be opened right now. Your messages are safe."
        error={messages.error}
        onRetry={() => void messages.refetch()}
        retryLabel="Retry"
      />
    );
  } else if (messages.data.messages.length === 0) {
    body = <p className="py-10 text-center text-base text-text-3">No messages yet.</p>;
  } else {
    body = <MessageList messages={messages.data.messages} currentUserId={user?.id} />;
  }

  return (
    <>
      <header className="flex h-header-mobile flex-none items-center gap-3 border-b border-border bg-surface pr-3 pl-2 desk:h-header desk:pl-6">
        <Link
          to="/chats"
          aria-label="Back to chats"
          className="flex size-touch flex-none items-center justify-center rounded-md text-text-2 desk:hidden"
        >
          <Icon name="back" />
        </Link>
        {chat.data ? (
          <Avatar
            seed={chat.data.id}
            name={title}
            size="md"
            shape={chat.data.type === "group" ? "square" : "round"}
          />
        ) : (
          <div className="size-avatar-md flex-none animate-pulse rounded-md bg-skeleton" />
        )}
        <div className="min-w-0 flex-1">
          <h1 className="m-0 truncate text-[16px] leading-[22px] font-semibold">{title}</h1>
          {chat.data && <div className="text-sm text-text-3">{chatSubtitle(chat.data)}</div>}
        </div>
      </header>
      <div
        aria-busy={messages.isPending}
        className="flex min-h-0 flex-1 [scrollbar-width:thin] flex-col-reverse overflow-y-auto"
      >
        <div className="mx-auto w-full max-w-content px-4 pb-3 desk:px-6">{body}</div>
      </div>
    </>
  );
}
