import type { ButtonHTMLAttributes, ReactNode } from "react";

const variants = {
  primary: "bg-accent text-accent-fg hover:bg-accent-hover",
  secondary: "border border-border-input bg-surface text-text hover:bg-row-hover",
} as const;

const heights = {
  md: "h-touch",
  lg: "h-12",
} as const;

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof variants;
  size?: keyof typeof heights;
  icon?: ReactNode;
  fullWidth?: boolean;
}

export function Button({
  variant = "primary",
  size = "md",
  icon,
  fullWidth = true,
  className,
  children,
  type = "button",
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={`inline-flex items-center justify-center gap-2 rounded-md px-[18px] text-md leading-5 font-semibold whitespace-nowrap transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${fullWidth ? "w-full" : ""} ${variants[variant]} ${heights[size]} ${className ?? ""}`}
      {...rest}
    >
      {icon}
      {children}
    </button>
  );
}
