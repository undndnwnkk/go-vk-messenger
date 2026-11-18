import { Link } from "react-router";

import { StatePanel } from "../components/StatePanel";

export function NotFoundPage() {
  return (
    <div className="flex h-full flex-col bg-bg-app">
      <StatePanel
        visual={null}
        title="Page not found"
        description="The page you're looking for doesn't exist."
      >
        <Link
          to="/chats"
          className="inline-flex h-touch items-center justify-center rounded-md bg-accent font-semibold text-accent-fg no-underline"
        >
          Go to chats
        </Link>
      </StatePanel>
    </div>
  );
}
