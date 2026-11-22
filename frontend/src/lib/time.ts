const DAY_MS = 24 * 60 * 60 * 1000;

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

function daysBetween(date: Date, now: Date): number {
  return Math.round((startOfDay(now) - startOfDay(date)) / DAY_MS);
}

const clock = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit" });
const weekday = new Intl.DateTimeFormat("en-US", { weekday: "short" });
const longWeekday = new Intl.DateTimeFormat("en-US", { weekday: "long" });
const monthDay = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" });

export function formatClock(date: Date): string {
  return clock.format(date);
}

export function formatListTime(date: Date, now: Date = new Date()): string {
  const days = daysBetween(date, now);
  if (days <= 0) {
    return formatClock(date);
  }
  if (days === 1) {
    return "Yesterday";
  }
  if (days < 7) {
    return weekday.format(date);
  }
  return monthDay.format(date);
}

export function formatDayChip(date: Date, now: Date = new Date()): string {
  const days = daysBetween(date, now);
  if (days <= 0) {
    return "Today";
  }
  if (days === 1) {
    return "Yesterday";
  }
  return `${longWeekday.format(date)}, ${monthDay.format(date)}`;
}

export function isSameDay(a: Date, b: Date): boolean {
  return startOfDay(a) === startOfDay(b);
}
