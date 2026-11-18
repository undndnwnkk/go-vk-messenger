import { useId, type InputHTMLAttributes, type ReactNode } from "react";

import { Icon } from "./Icon";

interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "id" | "prefix"> {
  label: string;
  error?: string | undefined;
  hint?: string | undefined;
  prefix?: ReactNode;
  trailing?: ReactNode;
}

export function TextField({ label, error, hint, prefix, trailing, ...inputProps }: TextFieldProps) {
  const id = useId();
  const messageId = `${id}-message`;
  const message = error ?? hint;
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium text-text-2">
        {label}
      </label>
      <div
        className={`flex h-12 items-center gap-1.5 rounded-md border field-focus bg-surface pr-1 pl-3.5 ${error ? "border-danger" : "border-border-input"}`}
      >
        {prefix !== undefined && <span className="-mr-1 text-md text-text-3">{prefix}</span>}
        <input
          id={id}
          aria-invalid={error ? true : undefined}
          aria-describedby={message ? messageId : undefined}
          className="h-11 min-w-0 flex-1 border-0 bg-transparent text-md outline-none"
          {...inputProps}
        />
        {trailing}
      </div>
      {error ? (
        <div id={messageId} className="flex items-center gap-1.5 text-sm text-danger">
          <Icon name="alert" size={16} />
          <span>{error}</span>
        </div>
      ) : (
        hint && (
          <div id={messageId} className="text-sm text-text-3">
            {hint}
          </div>
        )
      )}
    </div>
  );
}
