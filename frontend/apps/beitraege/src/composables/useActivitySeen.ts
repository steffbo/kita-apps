import { useAuthStore } from '@/stores/auth';

// "New" parent changes are counted per browser and account since the changes page was last opened.
function key(): string {
  return `kita-activity-seen:${useAuthStore().user?.id ?? 'anon'}`;
}

/** Timestamp (ms) of the last visit to the changes page; 0 if never visited. */
export function activitySeenAt(): number {
  return Number(localStorage.getItem(key()) ?? 0) || 0;
}

export function markActivitySeen(): void {
  localStorage.setItem(key(), String(Date.now()));
}
