<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { AlertTriangle, X } from 'lucide-vue-next';
import type { ChildEinstufungSync } from '@/api/types';
import { formatCurrency, formatMonthYear } from '@/utils/format';

const props = defineProps<{ sync: ChildEinstufungSync }>();
defineEmits<{ close: [] }>();

const changes = computed(() => props.sync.expectationChanges);
const feeSummary = computed(() => {
  const c = changes.value;
  const parts = [
    c.created && `${c.created} angelegt`,
    c.updated && `${c.updated} aktualisiert`,
    c.removed && `${c.removed} entfernt`,
  ].filter(Boolean);
  return parts.length ? `Platzgeld: ${parts.join(', ')}.` : 'Das erzeugte Platzgeld war bereits passend.';
});
</script>

<template>
  <section
    data-testid="einstufung-sync"
    aria-labelledby="einstufung-sync-heading"
    class="note note-warning mb-6"
  >
    <AlertTriangle />
    <div class="min-w-0 flex-1">
      <div class="flex items-start justify-between gap-3">
      <h2 id="einstufung-sync-heading" class="font-semibold">
        Einstufung durch die Änderung angepasst
      </h2>
      <button type="button" class="p-1 rounded hover:bg-warning-soft" aria-label="Hinweis schließen" @click="$emit('close')">
        <X class="h-4 w-4" />
      </button>
      </div>
    <p class="mt-1 text-warning/90">
      Die geänderten Kinddaten verändern die Monatsbeträge der Einstufung. {{ feeSummary }}
    </p>
    <ul class="mt-2 space-y-1">
      <li v-for="e in sync.einstufungen" :key="e.id" class="flex flex-wrap items-center gap-x-2">
        <RouterLink :to="`/einstufungen/${e.id}`" class="font-medium text-primary hover:underline">
          Einstufung {{ e.year }}
        </RouterLink>
        <span v-if="e.letterSent" class="text-warning">
          – der versendete Brief ist nicht mehr aktuell, bitte neu senden.
        </span>
      </li>
    </ul>
    <div v-if="changes.creditReviewRequired.length" class="mt-2 text-warning">
      Bereits bezahlt, Gutschrift prüfen:
      <span v-for="(item, i) in changes.creditReviewRequired" :key="item.feeId">
        {{ i ? ', ' : '' }}{{ formatMonthYear(item.year, item.month) }} ({{ formatCurrency(item.creditAmount) }})
      </span>
    </div>
    </div>
  </section>
</template>
