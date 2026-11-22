import { useId, useState } from "react";
import { useNavigate } from "react-router";

import { Dialog } from "../../../components/Dialog";
import { IconButton } from "../../../components/IconButton";
import { DirectChatPane } from "./DirectChatPane";
import { GroupChatPane } from "./GroupChatPane";

export type NewChatMode = "direct" | "group";

interface NewChatDialogProps {
  mode: NewChatMode | null;
  onClose: () => void;
}

function segmentClass(active: boolean): string {
  const base = "flex h-9 flex-1 items-center justify-center rounded-[9px] text-base";
  return active
    ? `${base} bg-surface font-semibold text-text shadow-1`
    : `${base} font-medium text-text-2 hover:text-text`;
}

interface NewChatContentProps {
  initialMode: NewChatMode;
  titleId: string;
  onClose: () => void;
}

function NewChatContent({ initialMode, titleId, onClose }: NewChatContentProps) {
  const navigate = useNavigate();
  const [mode, setMode] = useState<NewChatMode>(initialMode);
  const title = mode === "direct" ? "New chat" : "New group";

  const onCreated = (chatId: string) => {
    onClose();
    void navigate(`/chats/${chatId}`);
  };

  return (
    <>
      <header className="flex flex-none items-center gap-2 pt-3 pr-3 pb-2 pl-6">
        <h2 id={titleId} className="m-0 flex-1 font-heading text-xl font-semibold">
          {title}
        </h2>
        <IconButton label="Close" icon="close" onClick={onClose} />
      </header>
      <div className="px-6 pt-1 pb-3">
        <div role="group" aria-label="Chat type" className="flex gap-1 rounded-md bg-sunken p-1">
          {(["direct", "group"] as const).map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={mode === value}
              className={segmentClass(mode === value)}
              onClick={() => {
                setMode(value);
              }}
            >
              {value === "direct" ? "Direct" : "Group"}
            </button>
          ))}
        </div>
      </div>
      {mode === "direct" ? (
        <DirectChatPane onCreated={onCreated} />
      ) : (
        <GroupChatPane onCreated={onCreated} onCancel={onClose} />
      )}
    </>
  );
}

export function NewChatDialog({ mode, onClose }: NewChatDialogProps) {
  const titleId = useId();
  return (
    <Dialog open={mode !== null} labelledBy={titleId} onClose={onClose}>
      {mode !== null && <NewChatContent initialMode={mode} titleId={titleId} onClose={onClose} />}
    </Dialog>
  );
}
