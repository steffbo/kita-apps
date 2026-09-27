<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
const props = defineProps<{ topic?: string; referenceId?: string }>();
const emit = defineEmits<{ close: []; saved: [] }>();
const topic = ref(props.topic ?? 'GENERAL');
const message = ref('');
const error = ref('');
const saving = ref(false);
const labels: Record<string, string> = {
  GENERAL: 'Allgemein', CHILD: 'Kind', FEE: 'Beitrag',
  PARENT_WORK: 'Elternstunden', CONTACT: 'Kontaktdaten',
};
async function save() {
  saving.value = true; error.value = '';
  try {
    await api.createOwnReport({ topic: topic.value,
      referenceId: topic.value === props.topic ? props.referenceId : undefined,
      message: message.value.trim() });
    emit('saved');
  } catch (e) { error.value = e instanceof Error ? e.message : 'Meldung fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>
<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close')">
    <form role="dialog" aria-modal="true" aria-label="Fehler melden"
      class="w-full max-w-lg rounded-2xl bg-card p-5 shadow-xl sm:p-6" @submit.prevent="save">
      <h2 class="text-xl font-bold">Fehler melden</h2>
      <label class="mt-4 block text-sm font-medium">Thema
        <select v-model="topic" class="mt-1 w-full rounded-lg border px-3 py-2">
          <option v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</option>
        </select>
      </label>
      <p v-if="referenceId && topic === props.topic" class="mt-2 text-xs text-muted-foreground">
        Bezug: {{ labels[topic] }}
      </p>
      <label class="mt-4 block text-sm font-medium">Nachricht
        <textarea v-model="message" maxlength="2000" required rows="5"
          class="mt-1 w-full rounded-lg border px-3 py-2" />
      </label>
      <p class="text-right text-xs text-muted-foreground">{{ message.length }}/2000 Zeichen</p>
      <p v-if="error" role="alert" class="mt-2 text-sm text-red-700 dark:text-red-300">{{ error }}</p>
      <div class="mt-5 flex justify-end gap-2">
        <button type="button" class="rounded-lg border px-4 py-2" @click="emit('close')">Abbrechen</button>
        <button type="submit" :disabled="saving || !message.trim()"
          class="rounded-lg bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50">
          Meldung senden
        </button>
      </div>
    </form>
  </div>
</template>
