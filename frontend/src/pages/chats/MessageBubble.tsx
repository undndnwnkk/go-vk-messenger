import type { Message } from "../../api/types";
import { formatClock } from "../../lib/time";

interface MessageBubbleProps {
  message: Message;
  own: boolean;
}

export function MessageBubble({ message, own }: MessageBubbleProps) {
  const time = formatClock(new Date(message.created_at));
  const align = own ? "justify-end" : "justify-start";

  if (message.deleted) {
    return (
      <div className={`flex ${align}`}>
        <div className="rounded-lg border border-dashed border-border-input px-3.5 py-2 text-base text-text-3 italic">
          This message was deleted
        </div>
      </div>
    );
  }

  const bubble = own
    ? "bg-bubble-out text-bubble-out-text"
    : "border border-bubble-in-border bg-bubble-in text-text";
  const meta = own ? "text-bubble-out-meta" : "text-text-3";

  return (
    <div className={`flex ${align}`}>
      <div className={`max-w-[min(560px,85%)] rounded-lg px-3.5 py-2 ${bubble}`}>
        {message.reply_to && (
          <div
            className={`mb-1.5 truncate rounded-sm border-l-2 border-accent px-2 py-1 text-sm ${own ? "bg-quote-out" : "bg-quote-in"}`}
          >
            {message.reply_to.deleted ? "Message deleted" : message.reply_to.content}
          </div>
        )}
        <p className="m-0 text-md break-words whitespace-pre-wrap">{message.content}</p>
        <div className={`mt-0.5 text-right text-xs ${meta}`}>
          {message.edited && <span>edited </span>}
          <time dateTime={message.created_at}>{time}</time>
        </div>
      </div>
    </div>
  );
}
