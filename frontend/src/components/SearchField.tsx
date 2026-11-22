import type { InputHTMLAttributes } from "react";

import { Icon } from "./Icon";

interface SearchFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
  label: string;
}

export function SearchField({ label, className, ...inputProps }: SearchFieldProps) {
  return (
    <div
      className={`flex h-touch items-center gap-2 rounded-md border field-focus border-transparent bg-sunken px-3 text-text-3 ${className ?? ""}`}
    >
      <Icon name="search" size={18} />
      <input
        type="search"
        aria-label={label}
        className="h-10 min-w-0 flex-1 border-0 bg-transparent text-md text-text outline-none"
        {...inputProps}
      />
    </div>
  );
}
