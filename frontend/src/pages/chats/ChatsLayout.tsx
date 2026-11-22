import { useState } from "react";
import { Outlet, useMatch } from "react-router";

import { Icon } from "../../components/Icon";
import { Logo } from "../../components/Logo";
import { AccountFooter } from "./AccountFooter";
import { ChatList } from "./ChatList";

export function ChatsLayout() {
  const chatOpen = useMatch("/chats/:id") !== null;
  const [filter, setFilter] = useState("");
  return (
    <div className="flex h-full overflow-hidden bg-bg-app">
      <aside
        className={`${chatOpen ? "hidden desk:flex" : "flex"} w-full flex-none flex-col bg-surface desk:w-list desk:border-r desk:border-border`}
      >
        <header className="flex h-header-mobile flex-none items-center gap-2 pr-3 pl-5 desk:h-header">
          <Logo size="md" />
        </header>
        <div className="flex-none px-4 pb-2">
          <div className="flex h-touch items-center gap-2 rounded-md border field-focus border-transparent bg-sunken px-3 text-text-3">
            <Icon name="search" size={18} />
            <input
              type="search"
              aria-label="Search chats"
              placeholder="Search chats"
              value={filter}
              onChange={(event) => {
                setFilter(event.target.value);
              }}
              className="h-10 min-w-0 flex-1 border-0 bg-transparent text-md text-text outline-none"
            />
          </div>
        </div>
        <ChatList filter={filter} />
        <AccountFooter />
      </aside>
      <main
        className={`${chatOpen ? "flex" : "hidden desk:flex"} min-w-0 flex-1 flex-col bg-chat-bg`}
      >
        <Outlet />
      </main>
    </div>
  );
}
