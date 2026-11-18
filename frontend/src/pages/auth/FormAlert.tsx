import { Icon } from "../../components/Icon";

export function FormAlert({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-center gap-2 rounded-md bg-danger-soft px-3.5 py-3 text-base text-danger"
    >
      <Icon name="alert" size={18} />
      <span>{message}</span>
    </div>
  );
}
