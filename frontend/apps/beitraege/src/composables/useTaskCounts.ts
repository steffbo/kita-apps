import { ref } from 'vue';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import { activitySeenAt } from '@/composables/useActivitySeen';

// Open items behind „Aufgaben“, shared by sidebar, tab bar and the section tabs.
const bank = ref(0);
const reports = ref(0);
const changes = ref(0);

const ACTIVITY_LIMIT = 100;
const MIN_INTERVAL_MS = 30_000;
let loadedAt = 0;
let pending: Promise<void> | null = null;

async function load() {
  const auth = useAuthStore();
  if (!auth.canAccessFees) {
    bank.value = reports.value = changes.value = 0;
    return;
  }
  const [unmatched, open, activity] = await Promise.allSettled([
    api.getUnmatchedTransactions({ perPage: 1 }),
    auth.isAdmin ? api.getParentReports('OPEN') : Promise.resolve([]),
    auth.isAdmin ? api.getParentActivity(ACTIVITY_LIMIT) : Promise.resolve([]),
  ] as const);
  if (unmatched.status === 'fulfilled') bank.value = unmatched.value.total;
  if (open.status === 'fulfilled') reports.value = open.value.length;
  if (activity.status === 'fulfilled') {
    const seenAt = activitySeenAt();
    changes.value = activity.value.filter(item => new Date(item.at).getTime() > seenAt).length;
  }
}

/** Reloads the counts; without `force` at most every 30 seconds. */
export function refreshTaskCounts(force = false): Promise<void> {
  if (pending) return pending;
  if (!force && Date.now() - loadedAt < MIN_INTERVAL_MS) return Promise.resolve();
  loadedAt = Date.now();
  pending = load().catch(() => {}).finally(() => { pending = null; });
  return pending;
}

export function useTaskCounts() {
  return { bank, reports, changes };
}
