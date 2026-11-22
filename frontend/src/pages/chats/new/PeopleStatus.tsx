import { errorMessage } from "../../../api/errors";

interface PeopleStatusProps {
  isPending: boolean;
  error: unknown;
  isEmpty: boolean;
  isContacts: boolean;
}

export function PeopleStatus({ isPending, error, isEmpty, isContacts }: PeopleStatusProps) {
  let text: string | null = null;
  if (isPending) {
    text = "Searching…";
  } else if (error) {
    text = errorMessage(error);
  } else if (isEmpty) {
    text = isContacts
      ? "No contacts yet. Search by username to find people."
      : "No people match this username.";
  }
  if (text === null) {
    return null;
  }
  return (
    <p role="status" className="m-0 px-6 py-4 text-base text-text-3">
      {text}
    </p>
  );
}
