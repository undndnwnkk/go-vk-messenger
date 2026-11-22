import { useAuth } from "../../auth/useAuth";
import { Avatar } from "../../components/Avatar";
import { IconButton } from "../../components/IconButton";
import { useTheme } from "../../theme/useTheme";

export function AccountFooter() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const username = user?.username ?? "";
  return (
    <footer className="flex flex-none items-center gap-3 border-t border-border py-2.5 pr-2 pl-4">
      {user ? (
        <Avatar seed={user.id} name={username} size="md" />
      ) : (
        <div className="size-avatar-md flex-none animate-pulse rounded-full bg-skeleton" />
      )}
      <div className="min-w-0 flex-1">
        <div className="truncate text-base font-semibold">{username}</div>
        {user && <div className="truncate text-xs text-text-3">@{username}</div>}
      </div>
      <IconButton
        label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        icon={theme === "dark" ? "sun" : "moon"}
        onClick={toggleTheme}
      />
      <IconButton label="Sign out" icon="signOut" onClick={logout} />
    </footer>
  );
}
