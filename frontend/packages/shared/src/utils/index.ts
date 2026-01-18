/**
 * Format date to German locale
 */
export function formatDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });
}

/**
 * Format time to HH:mm
 */
export function formatTime(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Format duration in minutes to hours and minutes string
 */
export function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  
  if (hours === 0) {
    return `${mins} Min.`;
  }
  if (mins === 0) {
    return `${hours} Std.`;
  }
  return `${hours} Std. ${mins} Min.`;
}

/**
 * Get Monday of the week for a given date
 */
export function getWeekStart(date: Date): Date {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
  d.setDate(diff);
  d.setHours(0, 0, 0, 0);
  return d;
}

/**
 * Get Sunday of the week for a given date
 */
export function getWeekEnd(date: Date): Date {
  const start = getWeekStart(date);
  const end = new Date(start);
  end.setDate(end.getDate() + 6);
  return end;
}

/**
 * Get array of dates for a week starting from Monday
 */
export function getWeekDates(startDate: Date): Date[] {
  const monday = getWeekStart(startDate);
  const dates: Date[] = [];
  
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday);
    d.setDate(d.getDate() + i);
    dates.push(d);
  }
  
  return dates;
}

/**
 * Format ISO date string (YYYY-MM-DD)
 */
export function toISODateString(date: Date): string {
  return date.toISOString().split('T')[0];
}

/**
 * German weekday names
 */
export const WEEKDAYS = ['Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag', 'Sonntag'];
export const WEEKDAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];

/**
 * Get German weekday name
 */
export function getWeekdayName(date: Date, short = false): string {
  const day = date.getDay();
  const index = day === 0 ? 6 : day - 1; // Convert Sunday (0) to index 6
  return short ? WEEKDAYS_SHORT[index] : WEEKDAYS[index];
}
