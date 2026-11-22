import { formatDayChip, formatListTime } from "./time";

const now = new Date(2026, 10, 20, 15, 30);

describe("formatListTime", () => {
  test.each([
    [new Date(2026, 10, 20, 9, 5), "09:05"],
    [new Date(2026, 10, 19, 23, 59), "Yesterday"],
    [new Date(2026, 10, 16, 12, 0), "Mon"],
    [new Date(2026, 10, 13, 12, 0), "Nov 13"],
    [new Date(2025, 0, 2, 12, 0), "Jan 2"],
  ])("%s → %s", (date, expected) => {
    expect(formatListTime(date, now)).toBe(expected);
  });
});

describe("formatDayChip", () => {
  test("uses Today, Yesterday, then weekday with date", () => {
    expect(formatDayChip(new Date(2026, 10, 20, 1, 0), now)).toBe("Today");
    expect(formatDayChip(new Date(2026, 10, 19, 1, 0), now)).toBe("Yesterday");
    expect(formatDayChip(new Date(2026, 10, 16, 1, 0), now)).toBe("Monday, Nov 16");
  });
});
