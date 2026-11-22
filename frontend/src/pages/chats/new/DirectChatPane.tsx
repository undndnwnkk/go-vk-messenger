import { useState } from "react";

import { errorMessage } from "../../../api/errors";
import { useCreateDirectChat } from "../../../api/mutations";
import { Avatar } from "../../../components/Avatar";
import { SearchField } from "../../../components/SearchField";
import { FormAlert } from "../../auth/FormAlert";
import { PeopleStatus } from "./PeopleStatus";
import { SectionLabel } from "./SectionLabel";
import { usePeopleSearch } from "./usePeopleSearch";

export function DirectChatPane({ onCreated }: { onCreated: (chatId: string) => void }) {
  const [query, setQuery] = useState("");
  const people = usePeopleSearch(query);
  const createChat = useCreateDirectChat();
  const users = people.data ?? [];

  return (
    <>
      <div className="px-6 pb-3">
        <SearchField
          label="Search people"
          placeholder="Search by username"
          autoFocus
          data-autofocus
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
          }}
        />
      </div>
      {createChat.isError && (
        <div className="px-6 pb-2">
          <FormAlert message={errorMessage(createChat.error)} />
        </div>
      )}
      <SectionLabel>{people.isContacts ? "Contacts" : "People"}</SectionLabel>
      <div className="flex min-h-0 flex-1 [scrollbar-width:thin] flex-col overflow-y-auto pb-3">
        <PeopleStatus
          isPending={people.isPending}
          error={people.error}
          isEmpty={users.length === 0}
          isContacts={people.isContacts}
        />
        {users.map((user) => (
          <button
            key={user.id}
            type="button"
            disabled={createChat.isPending}
            onClick={() => {
              createChat.mutate(user.id, {
                onSuccess: (chat) => {
                  onCreated(chat.id);
                },
              });
            }}
            className="flex h-14 w-full flex-none items-center gap-3 px-6 text-left hover:bg-row-hover disabled:opacity-60"
          >
            <Avatar seed={user.id} name={user.username} size="md" />
            <div className="min-w-0 flex-1">
              <div className="truncate text-md leading-5 font-medium">{user.username}</div>
              <div className="truncate text-sm text-text-3">@{user.username}</div>
            </div>
          </button>
        ))}
      </div>
    </>
  );
}
