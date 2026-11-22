const palette = [
  "bg-av1-bg text-av1-fg",
  "bg-av2-bg text-av2-fg",
  "bg-av3-bg text-av3-fg",
  "bg-av4-bg text-av4-fg",
  "bg-av5-bg text-av5-fg",
  "bg-av6-bg text-av6-fg",
] as const;

const sizes = {
  xs: "size-avatar-xs text-[10px]",
  md: "size-avatar-md text-[14px]",
  lg: "size-avatar-lg text-[16px]",
} as const;

function paletteFor(seed: string): string {
  let hash = 0;
  for (const char of seed) {
    hash = (hash * 31 + (char.codePointAt(0) ?? 0)) >>> 0;
  }
  return palette[hash % palette.length] ?? palette[0];
}

function initialsOf(name: string): string {
  const [first = "", second] = name
    .trim()
    .split(/[\s_.-]+/)
    .filter(Boolean);
  const letters = second === undefined ? first.slice(0, 2) : first.charAt(0) + second.charAt(0);
  return letters.toUpperCase();
}

interface AvatarProps {
  seed: string;
  name: string;
  shape?: "round" | "square";
  size?: keyof typeof sizes;
}

export function Avatar({ seed, name, shape = "round", size = "lg" }: AvatarProps) {
  const radius =
    shape === "round" ? "rounded-full" : size === "lg" ? "rounded-[14px]" : "rounded-md";
  return (
    <div
      aria-hidden="true"
      className={`flex flex-none items-center justify-center font-heading font-semibold tracking-[0.02em] ${sizes[size]} ${radius} ${paletteFor(seed)}`}
    >
      {initialsOf(name)}
    </div>
  );
}
