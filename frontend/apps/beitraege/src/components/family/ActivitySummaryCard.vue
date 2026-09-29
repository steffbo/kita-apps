<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { History, ChevronRight } from 'lucide-vue-next';
import { api } from '@/api';
import { activitySeenAt } from '@/composables/useActivitySeen';

const LIMIT = 100;
const newCount = ref<number | null>(null);

onMounted(async () => {
  try {
    const seenAt = activitySeenAt();
    const activity = await api.getParentActivity(LIMIT);
    newCount.value = activity.filter((item) => new Date(item.at).getTime() > seenAt).length;
  } catch {
    newCount.value = null;
  }
});

const label = computed(() => {
  const n = newCount.value;
  if (n === null) return 'Änderungen von Eltern';
  if (n === 0) return 'Keine neuen Änderungen von Eltern';
  const count = n >= LIMIT ? `${LIMIT}+` : String(n);
  return `${count} ${n === 1 ? 'neue Änderung' : 'neue Änderungen'} von Eltern`;
});
</script>

<template>
  <RouterLink
    to="/aenderungen"
    class="flex items-center gap-3 rounded-2xl border bg-card p-4 transition-colors hover:bg-accent"
  >
    <span
      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl"
      :class="newCount ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'"
    ><History class="h-5 w-5" /></span>
    <span class="flex-1 font-semibold text-foreground">{{ label }}</span>
    <ChevronRight class="h-5 w-5 text-muted-foreground" />
  </RouterLink>
</template>
