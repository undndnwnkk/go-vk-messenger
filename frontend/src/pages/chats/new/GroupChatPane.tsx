import { useState, type SubmitEvent } from "react";

import { errorMessage } from "../../../api/errors";
import { useCreateGroupChat } from "../../../api/mutations";
import type { UserSummary } from "../../../api/types";
import { Avatar } from "../../../components/Avatar";
import { Button } from "../../../components/Button";
import { Icon } from "../../../components/Icon";
import { SearchField } from "../../../components/SearchField";
import { TextField } from "../../../components/TextField";
import { FormAlert } from "../../auth/FormAlert";
import { PeopleStatus } from "./PeopleStatus";
import { usePeopleSearch } from "./usePeopleSearch";

const MAX_TITLE_LENGTH = 100;

interface GroupChatPaneProps {
  onCreated: (chatId: string) => void;
  onCancel: () => void;
}

export function GroupChatPane({ onCreated, onCancel }: GroupChatPaneProps) {
  const [title, setTitle] = useState("");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<UserSummary[]>([]);
  const people = usePeopleSearch(query);
  const createGroup = useCreateGroupChat();
  const users = people.data ?? [];
  const selectedIds = new Set(selected.map((user) => user.id));

  const toggle = (user: UserSummary) => {
    setSelected((current) =>
      current.some((item) => item.id === user.id)
        ? current.filter((item) => item.id !== user.id)
        : [...current, user],
    );
  };

  const onSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    createGroup.mutate(
      { title: title.trim(), userIds: selected.map((user) => user.id) },
      {
        onSuccess: (chat) => {
          onCreated(chat.id);
        },
      },
    );
  };

  const count = selected.length;

  return (
    <form className="flex min-h-0 flex-1 flex-col" onSubmit={onSubmit} noValidate>
      <div className="flex flex-col gap-3 px-6 pt-1 pb-3">
        {createGroup.isError && <FormAlert message={errorMessage(createGroup.error)} />}
        <TextField
          label="Group name"
          name="title"
          autoFocus
          data-autofocus
          required
          maxLength={MAX_TITLE_LENGTH}
          value={title}
          onChange={(event) => {
            setTitle(event.target.value);
          }}
        />
      </div>
      {count > 0 && (
        <ul aria-label="Selected members" className="m-0 flex list-none flex-wrap gap-2 px-6 pb-3">
          {selected.map((user) => (
            <li key={user.id}>
              <button
                type="button"
                aria-label={`Remove ${user.username}`}
                onClick={() => {
                  toggle(user);
                }}
                className="inline-flex h-9 items-center gap-2 rounded-[18px] bg-accent-soft pr-2.5 pl-1.5 text-base font-medium text-on-accent-soft"
              >
                <Avatar seed={user.id} name={user.username} size="xs" />
                <span>{user.username}</span>
                <Icon name="close" size={16} />
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="px-6 pb-2">
        <SearchField
          label="Add members"
          placeholder="Add members"
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
          }}
        />
      </div>
      <div className="flex min-h-0 flex-1 [scrollbar-width:thin] flex-col overflow-y-auto pb-2">
        <PeopleStatus
          isPending={people.isPending}
          error={people.error}
          isEmpty={users.length === 0}
          isContacts={people.isContacts}
        />
        {users.map((user) => (
          <label
            key={user.id}
            className="flex h-14 w-full flex-none cursor-pointer items-center gap-3 px-6 hover:bg-row-hover"
          >
            <Avatar seed={user.id} name={user.username} size="md" />
            <div className="min-w-0 flex-1">
              <div className="truncate text-md leading-5 font-medium">{user.username}</div>
              <div className="truncate text-sm text-text-3">@{user.username}</div>
            </div>
            <input
              type="checkbox"
              aria-label={`Select ${user.username}`}
              checked={selectedIds.has(user.id)}
              onChange={() => {
                toggle(user);
              }}
              className="m-0 size-5 flex-none accent-accent"
            />
          </label>
        ))}
      </div>
      <footer className="flex flex-none items-center gap-3 border-t border-border px-6 py-4">
        <span className="flex-1 text-sm text-text-2">
          {count === 1 ? "1 member selected" : `${count} members selected`}
        </span>
        <Button variant="secondary" fullWidth={false} onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type="submit"
          fullWidth={false}
          disabled={title.trim() === "" || createGroup.isPending}
        >
          {createGroup.isPending ? "Creating…" : "Create group"}
        </Button>
      </footer>
    </form>
  );
}
