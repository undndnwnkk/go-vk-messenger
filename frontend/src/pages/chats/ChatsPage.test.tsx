import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";

import { API_URL } from "../../api/client";
import type { Chat } from "../../api/types";
import { writeToken } from "../../auth/tokenStore";
import { alice, bob, makeChat, me } from "../../test/fixtures";
import { renderApp } from "../../test/renderApp";
import { server } from "../../test/server";

const now = new Date().toISOString();

const groupChat = makeChat({
  id: "44444444-4444-4444-8444-444444444444",
  type: "group",
  title: "Platform team",
  unread_count: 3,
  last_message: {
    id: 9,
    sender_id: bob.id,
    sender_username: "bob",
    content: "Deploy is green",
    deleted: false,
    created_at: now,
  },
});

const directChat = makeChat({
  id: "55555555-5555-4555-8555-555555555555",
  type: "direct",
  peer: alice,
  muted: true,
  unread_count: 1,
  last_message: {
    id: 8,
    sender_id: alice.id,
    sender_username: "alice",
    content: "",
    deleted: true,
    created_at: now,
  },
});

function useApi(chats: Chat[]) {
  server.use(
    http.get(`${API_URL}/api/v1/me`, () => HttpResponse.json(me)),
    http.get(`${API_URL}/api/v1/chats`, () => HttpResponse.json(chats)),
    http.get(`${API_URL}/api/v1/users/search`, ({ request }) => {
      const q = new URL(request.url).searchParams.get("q") ?? "";
      const people = q === "" ? [bob] : [alice, bob].filter((user) => user.username.startsWith(q));
      return HttpResponse.json(people);
    }),
    http.get(`${API_URL}/api/v1/chats/:id`, ({ params }) =>
      HttpResponse.json(makeChat({ id: String(params.id), type: "direct", peer: alice })),
    ),
    http.get(`${API_URL}/api/v1/chats/:id/messages`, () => HttpResponse.json({ messages: [] })),
  );
}

beforeEach(() => {
  writeToken("valid-jwt", true);
});

test("chat list shows titles, previews, unread and muted state", async () => {
  useApi([groupChat, directChat]);
  renderApp("/chats");

  const list = await screen.findByRole("navigation", { name: "Chats" });
  const rows = await within(list).findAllByRole("link");
  expect(rows.map((row) => row.getAttribute("href"))).toEqual([
    `/chats/${groupChat.id}`,
    `/chats/${directChat.id}`,
  ]);
  const [groupRow, directRow] = rows as [HTMLElement, HTMLElement];
  expect(groupRow).toHaveTextContent("Platform team");
  expect(groupRow).toHaveTextContent("bob: Deploy is green");
  expect(groupRow).toHaveTextContent("Unread messages: 3");
  expect(directRow).toHaveTextContent("alice");
  expect(directRow).toHaveTextContent("Message deleted");
  expect(directRow).toHaveTextContent("Muted");
});

test("starting a direct chat searches people and opens the new chat", async () => {
  useApi([]);
  let createdWith: unknown = null;
  server.use(
    http.post(`${API_URL}/api/v1/chats/direct`, async ({ request }) => {
      createdWith = await request.json();
      return HttpResponse.json(
        makeChat({ id: "66666666-6666-4666-8666-666666666666", type: "direct", peer: alice }),
      );
    }),
  );
  const user = userEvent.setup();
  const { router } = renderApp("/chats");

  await user.click(await screen.findByRole("button", { name: "New chat" }));
  const dialog = await screen.findByRole("dialog", { name: "New chat" });
  expect(await within(dialog).findByText("Contacts")).toBeInTheDocument();
  expect(await within(dialog).findByText("@bob")).toBeInTheDocument();

  const search = within(dialog).getByRole("searchbox", { name: "Search people" });
  expect(search).toHaveFocus();
  await user.type(search, "al");
  await user.click(await within(dialog).findByRole("button", { name: /alice/ }));

  await waitFor(() => {
    expect(router.state.location.pathname).toBe("/chats/66666666-6666-4666-8666-666666666666");
  });
  expect(createdWith).toEqual({ user2_id: alice.id });
  await waitFor(() => {
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});

test("creating a group sends the title and selected members", async () => {
  useApi([]);
  let createdWith: unknown = null;
  server.use(
    http.post(`${API_URL}/api/v1/chats/group`, async ({ request }) => {
      createdWith = await request.json();
      return HttpResponse.json(
        makeChat({ id: "77777777-7777-4777-8777-777777777777", type: "group", title: "Q4" }),
        { status: 201 },
      );
    }),
  );
  const user = userEvent.setup();
  const { router } = renderApp("/chats");

  await user.click(await screen.findByRole("button", { name: "Create group" }));
  const dialog = await screen.findByRole("dialog", { name: "New group" });
  const create = within(dialog).getByRole("button", { name: "Create group" });
  expect(create).toBeDisabled();

  const groupName = within(dialog).getByRole("textbox", { name: "Group name" });
  expect(groupName).toHaveFocus();
  await user.type(groupName, "  Q4 ");
  await user.type(within(dialog).getByRole("searchbox", { name: "Add members" }), "a");
  await user.click(await within(dialog).findByRole("checkbox", { name: "Select alice" }));
  expect(within(dialog).getByText("1 member selected")).toBeInTheDocument();
  expect(within(dialog).getByRole("button", { name: "Remove alice" })).toBeInTheDocument();

  await user.click(create);

  await waitFor(() => {
    expect(router.state.location.pathname).toBe("/chats/77777777-7777-4777-8777-777777777777");
  });
  expect(createdWith).toEqual({ title: "Q4", user_ids: [alice.id] });
});
