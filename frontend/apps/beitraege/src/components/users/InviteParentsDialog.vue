<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { InvitationCandidate, InviteParentsResponse } from '@/api/types';

const emit = defineEmits<{ close: []; changed: [] }>();
const candidates = ref<InvitationCandidate[]>([]);
const selected = ref<string[]>([]);
const results = ref<InviteParentsResponse['results']>([]);
const loading = ref(true);
const sending = ref(false);
const error = ref('');
const selectable = computed(() => candidates.value.filter((item) => !item.ambiguous));
const allSelected = computed(() => selectable.value.length > 0 &&
  selectable.value.every((item) => selected.value.includes(item.parentId)));

async function load() {
  loading.value = true;
  error.value = '';
  try {
    candidates.value = await api.getInvitationCandidates();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Eltern konnten nicht geladen werden.';
  } finally {
    loading.value = false;
  }
}

function toggleAll() {
  selected.value = allSelected.value ? [] : selectable.value.map((item) => item.parentId);
}

async function submit() {
  if (!selected.value.length) return;
  sending.value = true;
  error.value = '';
  results.value = [];
  try {
    results.value = (await api.inviteParents(selected.value)).results;
    selected.value = [];
    emit('changed');
    await load();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Einladungen konnten nicht gesendet werden.';
  } finally {
    sending.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation">
    <div class="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-card p-6 shadow-xl"
      role="dialog" aria-modal="true" aria-labelledby="invite-title">
      <h2 id="invite-title" class="text-xl font-semibold">Eltern einladen</h2>
      <p class="mt-1 text-sm text-muted-foreground">
        Eltern mit aktivem Kind und ohne Benutzerkonto auswählen. Jede Einladung enthält einen Link zum
        Setzen des Passworts.
      </p>
      <p v-if="error" class="mt-4 text-sm text-destructive" role="alert">{{ error }}</p>
      <p v-if="loading" class="mt-4 text-sm text-muted-foreground">Eltern werden geladen...</p>
      <template v-else>
        <div v-if="!candidates.length" class="mt-4 text-sm text-muted-foreground">
          Keine Eltern mit aktivem Kind und E-Mail-Adresse ohne Benutzerkonto gefunden.
        </div>
        <div v-else class="mt-4 max-h-72 overflow-y-auto rounded-lg border">
          <label class="flex items-center gap-3 border-b bg-muted px-3 py-2 text-sm">
            <input type="checkbox" :checked="allSelected" :disabled="!selectable.length || sending"
              @change="toggleAll" /> Alle auswählen
          </label>
          <label v-for="item in candidates" :key="item.parentId"
            class="flex items-center gap-3 border-b px-3 py-2 text-sm last:border-b-0"
            :class="{ 'text-muted-foreground': item.ambiguous }">
            <input v-model="selected" type="checkbox" :value="item.parentId"
              :disabled="item.ambiguous || sending" />
            <span class="min-w-0 flex-1">
              <span class="font-medium">{{ item.firstName }} {{ item.lastName }}</span>
              <span class="ml-2 break-all">{{ item.email }}</span>
              <span class="block text-xs text-muted-foreground">
                {{ item.children.length === 1 ? 'Kind' : 'Kinder' }}: {{ item.children.join(', ') }}
              </span>
              <span v-if="item.ambiguous" class="block text-xs">
                Mehrere Elternteile haben diese E-Mail-Adresse.
              </span>
            </span>
          </label>
        </div>
      </template>
      <div v-if="results.length" class="mt-4 space-y-1" aria-live="polite">
        <p class="font-medium">Ergebnis</p>
        <p v-for="(result, index) in results" :key="index" class="text-sm">
          {{ result.email || result.parentId }}:
          <span :class="result.success ? 'text-green-700 dark:text-green-300' : 'text-destructive'">
            {{ result.success ? 'Einladung gesendet' : result.error }}
          </span>
        </p>
      </div>
      <div class="mt-6 flex justify-end gap-2">
        <button type="button" class="rounded-lg border px-4 py-2" @click="emit('close')">Schließen</button>
        <button type="button" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground
          disabled:opacity-50" :disabled="!selected.length || sending" @click="submit">
          {{ sending ? 'Einladungen werden gesendet...' : 'Ausgewählte einladen' }}
        </button>
      </div>
    </div>
  </div>
</template>
