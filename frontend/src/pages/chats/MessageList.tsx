import { Fragment } from "react";

import type { Message } from "../../api/types";
import { formatDayChip, isSameDay } from "../../lib/time";
import { MessageBubble } from "./MessageBubble";

interface MessageListProps {
  messages: Message[];
  currentUserId: string | undefined;
}

export function MessageList({ messages, currentUserId }: MessageListProps) {
  const chronological = messages.toReversed();
  return (
    <ol className="m-0 flex list-none flex-col gap-1 p-0">
      {chronological.map((message, index) => {
        const created = new Date(message.created_at);
        const previous = chronological[index - 1];
        const newDay = previous === undefined || !isSameDay(new Date(previous.created_at), created);
        return (
          <Fragment key={message.id}>
            {newDay && (
              <li className="mt-5 mb-2 flex justify-center">
                <span className="rounded-pill bg-sunken px-3 py-1 text-xs font-medium text-text-2">
                  {formatDayChip(created)}
                </span>
              </li>
            )}
            <li>
              <MessageBubble message={message} own={message.sender_id === currentUserId} />
            </li>
          </Fragment>
        );
      })}
    </ol>
  );
}
