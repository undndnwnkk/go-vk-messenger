# Plover — web messenger: design handoff

Design for a web messenger (UI language: English). Light and dark themes, desktop and mobile.
Everything here was exported from a design canvas; the HTML files are **visual references**, not production code.

## What is in this folder

- `tokens.css` — the single source of truth for colors, fonts, type scale, spacing, radii, sizes, shadows. Both themes. Copy it into the app as-is.
- `screens/*.html` — one static page per screen, open in a browser. `screens/index.html` links to all of them.
  Names: `desktop-<screen>-<light|dark>.html` and `mobile-<screen>-<light|dark>.html`.
  Start with `desktop-tokens-board.html` and `desktop-components-board.html`.
- Styling in the references is inline and uses `var(--token)`; rebuild it as real components and keep the token names.

## Screens

| Screen | Desktop file | Mobile file |
|---|---|---|
| Sign in | `desktop-login-*` | `mobile-login-*` |
| Create account | `desktop-register-*` | — |
| Main: chat list + open chat | `desktop-main-*` | `mobile-list-*`, `mobile-chat-*` |
| Message context menu + reply composer | `desktop-menu-*` | `mobile-menu-*` (bottom sheet) |
| In-chat search | `desktop-search-*` | `mobile-search-*` |
| Group members / group info | `desktop-members-*` | `mobile-members-*` |
| New direct chat | `desktop-new-direct-*` | `mobile-new-direct-*` |
| New group chat | `desktop-new-group-*` | `mobile-new-group-*` |
| Empty list | `desktop-empty-*` | `mobile-empty-*` |
| Loading (skeleton) | `desktop-loading-*` | `mobile-loading-*` |
| Error (list and chat failed to load) | `desktop-error-*` | `mobile-error-*` |
| Reconnecting banner | `desktop-reconnecting-*` | `mobile-reconnecting-*` |

## Theming

Put `class="pl-light"` or `class="pl-dark"` on the root element. Components use only `var(--token)`; never hard-code colors.
Respect `prefers-color-scheme` for the default and let the user override it (the theme toggle sits in the account row at the bottom of the chat list).

## Layout

- Desktop: two columns. Chat list 340px (`--list-w`), chat area flexible, optional right panel 360px (`--panel-w`) for group info. Message column max 760px (`--content-max`), centered.
- Mobile (< 760px): list and chat are separate screens, with a back button in the chat header. Group info, new chat, and new group are full screens. Message actions open as a bottom sheet.
- Touch targets are at least 44px (`--touch`). Icon buttons all have `aria-label`.

## Behavior to implement

**Chat list row**: avatar (round for direct, rounded-square for group), title, last message preview (prefix `You:` or sender name in groups), time, unread badge, mute icon.
Time format: `HH:mm` today, `Yesterday`, weekday within the week, otherwise `Mon D`. Muted chats show a gray badge instead of the accent one. A deleted last message shows italic `Message deleted`.

**Messages**
- Grouped by day with a centered date chip (`Today`, `Yesterday`, weekday + date).
- Consecutive messages from the same sender are clustered: sender name only above the first one, avatar only beside the last one (groups only), tighter corner radii inside a cluster.
- Own messages on the right, accent-soft bubble; others on the left, white bubble with border.
- Statuses on own messages: `sending` (clock icon), `sent` (check icon), `failed` (alert icon + "Not sent" + Retry). `edited` adds the word "edited" before the time. `deleted` renders a dashed placeholder "This message was deleted" and drops text, reactions and reply.
- Reply: quoted block at the top of the bubble (author + one-line snippet). While composing a reply or an edit, a strip above the input shows what is being replied to or edited, with a cancel button.
- Reactions: chips under the bubble with emoji and count; the current user's reactions are highlighted. Tapping a chip toggles the user's reaction.
- Context menu (right-click on desktop, long-press on mobile): quick reactions row, Reply, React…, Edit (own messages only), Delete (own messages, destructive color).
- Search in chat: header switches to a search field with "N of M" counter and previous/next buttons; matches are highlighted, the current match is stronger.
- Composer: attach, text input, emoji, send. Send is disabled while the input is empty.

**States**
- Loading: skeleton rows in the list and skeleton bubbles in the chat; list container has `aria-busy="true"`.
- Empty list: illustration, "No chats yet", buttons New chat and Create group.
- Error: list and chat each show their own message with a Retry button.
- Reconnecting: full-width warning banner at the top of the app ("Reconnecting… Messages will send when you are back online."), role `status`. The composer stays enabled; queued messages show the `sending` status. Remove the banner when the connection returns.

**Auth**: Sign in / Create account switch, inline field errors (e.g. "That username is taken."), show-password toggle, "Keep me signed in".

**Group info**: avatar, name, member count, mute toggle, members list with roles (Owner, Admin, Member), Add members, per-member actions menu (not for the owner or yourself), Leave group.

## Data the UI needs (frontend view of the model)

- `User`: `id`, `displayName`, `handle`, `avatarUrl?`
- `Chat`: `id`, `type` (`direct` | `group`), `title`, `avatarUrl?`, `lastMessage` (author, text snippet or deleted flag, `createdAt`), `unreadCount`, `muted`
- `Member`: `user`, `role` (`owner` | `admin` | `member`)
- `Message`: `id`, `chatId`, `author`, `text`, `createdAt`, `editedAt?`, `deletedAt?`, `replyTo?` (message id + author + snippet), `reactions[]` (`emoji`, `count`, `reactedByMe`), local-only `status` (`sending` | `sent` | `failed`)
- Realtime connection state: `connected` | `reconnecting`

## Suggested prompt for Claude Code

> Build the frontend for the messenger described in this folder. Read `README.md`, `tokens.css`, and open the files in `screens/` to see every screen in both themes. Use `tokens.css` unchanged and style only with its variables. Implement the desktop and mobile layouts, light/dark theme switching, and all states (loading, empty, error, reconnecting). Use mock data and a mock realtime layer first, behind an interface that can later be replaced by the real backend API. Stack: [YOUR STACK HERE].
