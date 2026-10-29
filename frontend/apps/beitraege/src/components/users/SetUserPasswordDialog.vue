<script setup lang="ts">
import { ref } from 'vue';
import { KeyRound, Loader2 } from 'lucide-vue-next';
import { api } from '@/api';
import type { UserAccount } from '@/api/types';
import AppSheet from '@/components/ui/AppSheet.vue';

const MIN_LENGTH = 8;

const props = defineProps<{ user: UserAccount }>();
const emit = defineEmits<{ close: []; saved: [] }>();

const password = ref('');
const isSaving = ref(false);
const error = ref<string | null>(null);

async function submit() {
  error.value = null;
  if (password.value.length < MIN_LENGTH) {
    error.value = `Das Passwort muss mindestens ${MIN_LENGTH} Zeichen haben.`;
    return;
  }
  isSaving.value = true;
  try {
    await api.setUserPassword(props.user.id, password.value);
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Passwort konnte nicht gesetzt werden';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet title="Passwort neu setzen" size="sm" @close="emit('close')">
    <form id="user-password-form" class="space-y-4" @submit.prevent="submit">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <KeyRound class="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 class="text-xl font-semibold">Passwort neu setzen</h2>
          <p class="text-sm text-muted-foreground">{{ user.email }} wird auf allen Geräten abgemeldet.</p>
        </div>
      </div>

      <label class="block">
        <span class="text-sm font-medium text-foreground">Neues Passwort (mind. {{ MIN_LENGTH }} Zeichen)</span>
        <input v-model="password" type="text" required autocomplete="off" class="mt-1 w-full input font-mono" />
      </label>

      <div v-if="error" class="note note-danger" role="alert">
        {{ error }}
      </div>

    </form>
    <template #footer>
        <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
        <button type="submit" form="user-password-form" :disabled="isSaving" class="btn btn-primary">
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          Passwort setzen
        </button>
    </template>
  </AppSheet>
</template>
