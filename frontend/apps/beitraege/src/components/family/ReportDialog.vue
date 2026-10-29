<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import AppSheet from '@/components/ui/AppSheet.vue';
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
  <AppSheet title="Fehler melden" size="md" @close="emit('close')">
    <form id="report-form" class="space-y-4" @submit.prevent="save">
      <label class="label">Thema
        <select v-model="topic" class="input">
          <option v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</option>
        </select>
      </label>
      <p v-if="referenceId && topic === props.topic" class="mt-2 text-xs text-muted-foreground">
        Bezug: {{ labels[topic] }}
      </p>
      <label class="label">Nachricht
        <textarea v-model="message" maxlength="2000" required rows="5"
          class="input" />
      </label>
      <p class="text-right text-xs text-muted-foreground">{{ message.length }}/2000 Zeichen</p>
      <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="report-form" :disabled="saving || !message.trim()"
        class="btn btn-primary">Meldung senden</button>
    </template>
  </AppSheet>
</template>
