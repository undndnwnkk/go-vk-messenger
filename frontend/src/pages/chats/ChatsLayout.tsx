import { useState } from "react";
import { Outlet, useMatch } from "react-router";

import { IconButton } from "../../components/IconButton";
import { Logo } from "../../components/Logo";
import { SearchField } from "../../components/SearchField";
import { AccountFooter } from "./AccountFooter";
import { ChatList } from "./ChatList";
import { NewChatDialog, type NewChatMode } from "./new/NewChatDialog";

export function ChatsLayout() {
  const chatOpen = useMatch("/chats/:id") !== null;
  const [filter, setFilter] = useState("");
  const [newChatMode, setNewChatMode] = useState<NewChatMode | null>(null);
  return (
    <div className="flex h-full overflow-hidden bg-bg-app">
      <aside
        className={`${chatOpen ? "hidden desk:flex" : "flex"} w-full flex-none flex-col bg-surface desk:w-list desk:border-r desk:border-border`}
      >
        <header className="flex h-header-mobile flex-none items-center gap-2 pr-3 pl-5 desk:h-header">
          <Logo size="md" />
          <span className="flex-1" />
          <IconButton
            label="New chat"
            icon="compose"
            onClick={() => {
              setNewChatMode("direct");
            }}
          />
        </header>
        <div className="flex-none px-4 pb-2">
          <SearchField
            label="Search chats"
            placeholder="Search chats"
            value={filter}
            onChange={(event) => {
              setFilter(event.target.value);
            }}
          />
        </div>
        <ChatList filter={filter} onNewChat={setNewChatMode} />
        <AccountFooter />
      </aside>
      <main
        className={`${chatOpen ? "flex" : "hidden desk:flex"} min-w-0 flex-1 flex-col bg-chat-bg`}
      >
        <Outlet />
      </main>
      <NewChatDialog
        mode={newChatMode}
        onClose={() => {
          setNewChatMode(null);
        }}
      />
    </div>
  );
}
