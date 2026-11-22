import type { ButtonHTMLAttributes } from "react";

import { Icon, type IconName } from "./Icon";

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "aria-label"> {
  label: string;
  icon: IconName;
}

export function IconButton({ label, icon, className, type = "button", ...rest }: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      className={`flex size-touch flex-none items-center justify-center rounded-md text-text-2 transition-colors hover:bg-row-hover disabled:opacity-50 ${className ?? ""}`}
      {...rest}
    >
      <Icon name={icon} />
    </button>
  );
}
