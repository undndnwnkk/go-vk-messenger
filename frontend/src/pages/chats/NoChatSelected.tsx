import { StatePanel } from "../../components/StatePanel";
import { Icon } from "../../components/Icon";

export function NoChatSelected() {
  return (
    <StatePanel
      visual={
        <div className="flex size-14 items-center justify-center rounded-full bg-accent-soft text-on-accent-soft">
          <Icon name="compose" size={26} />
        </div>
      }
      title="Select a chat"
      description="Choose a conversation from the list to start messaging."
    />
  );
}
