<script setup lang="ts">
import { computed, toRefs } from 'vue';
import type { EmailLog } from '@/api/types';
import { Eye } from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';
import { formatEmailType } from '@/utils/reminders';

const props = defineProps<{
  chronology: EmailLog[];
  isChronologyLoading: boolean;
  selectedChronologyLog: EmailLog | null;
}>();

const {
  chronology,
  isChronologyLoading,
} = toRefs(props);

const emit = defineEmits<{
  'update:selectedChronologyLog': [value: EmailLog | null];
}>();
const selectedChronologyLog = computed({
  get: () => props.selectedChronologyLog,
  set: (value) => emit('update:selectedChronologyLog', value),
});
</script>

<template>
  <div class="mt-6 pt-4 border-t">
    <h3 class="text-sm font-semibold text-foreground mb-2">Chronik dieser Familie</h3>
    <div v-if="isChronologyLoading" class="text-xs text-muted-foreground">Wird geladen...</div>
    <div v-else-if="chronology.length === 0" class="text-xs text-muted-foreground">Noch keine E-Mails an diese Familie gesendet.</div>
    <ul v-else class="divide-y">
      <li v-for="log in chronology" :key="log.id" class="py-2 flex items-center justify-between gap-3 text-sm">
        <div class="min-w-0">
          <span class="text-foreground">{{ formatDateTime(log.sentAt) }}</span>
          <span class="text-muted-foreground"> · {{ formatEmailType(log.emailType) }}</span>
          <span class="block truncate text-muted-foreground">{{ log.subject }}</span>
        </div>
        <button class="text-primary hover:underline shrink-0 inline-flex items-center gap-1" @click="selectedChronologyLog = log">
          <Eye class="h-4 w-4" />
          Anzeigen
        </button>
      </li>
    </ul>
  </div>
</template>
