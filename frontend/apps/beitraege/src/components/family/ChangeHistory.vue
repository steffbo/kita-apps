<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { DataChange } from '@/api/types';
import { formatDateTime } from '@/utils/format';
const props = defineProps<{ kind: 'parent' | 'child'; id: string }>();
const open = ref(false);
const changes = ref<DataChange[]>([]);
const error = ref('');
const actors = ref<Record<string, string>>({});
const fields: Record<string, string> = {
  email: 'E-Mail', phone: 'Telefon', street: 'Straße', streetNo: 'Nr.',
  postalCode: 'PLZ', city: 'Ort', firstName: 'Vorname', lastName: 'Nachname',
  birthDate: 'Geburtsdatum',
};
async function toggle() {
  open.value = !open.value;
  if (!open.value) return;
  try {
    changes.value = props.kind === 'parent'
      ? await api.getParentChanges(props.id) : await api.getChildChanges(props.id);
    const parentIds = [...new Set(changes.value.flatMap(c => c.parentId ? [c.parentId] : []))];
    const [parents, users] = await Promise.all([
      Promise.all(parentIds.map(id => api.getParent(id).catch(() => null))),
      api.getUsers().catch(() => []),
    ]);
    for (const parent of parents) if (parent) actors.value[parent.id] =
      `${parent.firstName} ${parent.lastName}`;
    for (const user of users) actors.value[user.id] =
      `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim() || user.email;
  } catch (e) { error.value = e instanceof Error ? e.message : 'Verlauf konnte nicht geladen werden'; }
}
</script>
<template>
  <section class="rounded-xl border bg-card p-5">
    <button class="flex w-full justify-between text-left font-bold" :aria-expanded="open" @click="toggle">
      Änderungsverlauf <span>{{ open ? '−' : '+' }}</span>
    </button>
    <div v-if="open" class="mt-4 space-y-3 text-sm">
      <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
      <p v-else-if="!changes.length" class="text-muted-foreground">Noch keine Änderungen.</p>
      <div v-for="change in changes" :key="change.id" class="border-t pt-3">
        <p class="font-semibold">{{ fields[change.field] ?? change.field }}</p>
        <p>{{ change.oldValue || '—' }} → {{ change.newValue || '—' }}</p>
        <p class="text-muted-foreground">{{ change.parentId ? actors[change.parentId] || 'Elternteil'
            : actors[change.userId ?? ''] || 'Mitarbeiter' }} ·
          {{ formatDateTime(change.changedAt) }}</p>
      </div>
    </div>
  </section>
</template>
