import { alice, bob, makeChat, me } from "../test/fixtures";
import { chatPreview, chatSubtitle, chatTitle } from "./chat";

const lastMessage = {
  id: 7,
  content: "  see you\n tomorrow ",
  deleted: false,
  created_at: "2026-11-20T10:00:00Z",
};

describe("chat presentation", () => {
  test("direct chats are named after the other user, groups after their title", () => {
    const direct = makeChat({ id: "d", type: "direct", peer: alice });
    const group = makeChat({ id: "g", type: "group", title: "Platform team" });
    expect(chatTitle(direct)).toBe("alice");
    expect(chatSubtitle(direct)).toBe("@alice");
    expect(chatTitle(group)).toBe("Platform team");
    expect(chatSubtitle(group)).toBe("Group");
  });

  test.each([
    ["own message", "group", me, { kind: "text", prefix: "You", text: "see you tomorrow" }],
    ["other in group", "group", bob, { kind: "text", prefix: "bob", text: "see you tomorrow" }],
    ["other in direct", "direct", alice, { kind: "text", prefix: null, text: "see you tomorrow" }],
  ] as const)("preview of %s", (_, type, sender, expected) => {
    const chat = makeChat({
      id: "c",
      type,
      last_message: { ...lastMessage, sender_id: sender.id, sender_username: sender.username },
    });
    expect(chatPreview(chat, me.id)).toEqual(expected);
  });

  test("deleted and missing last messages", () => {
    const deleted = makeChat({
      id: "c",
      type: "group",
      last_message: {
        ...lastMessage,
        content: "",
        deleted: true,
        sender_id: bob.id,
        sender_username: "bob",
      },
    });
    expect(chatPreview(deleted, me.id)).toEqual({ kind: "deleted" });
    expect(chatPreview(makeChat({ id: "e", type: "group" }), me.id)).toEqual({ kind: "empty" });
  });
});
