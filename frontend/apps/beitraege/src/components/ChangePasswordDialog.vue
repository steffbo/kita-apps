<script setup lang="ts">
import { ref } from 'vue';
import { KeyRound, Loader2 } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const MIN_LENGTH = 8;

const emit = defineEmits<{ close: []; changed: [] }>();

const authStore = useAuthStore();
const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const isSaving = ref(false);
const error = ref<string | null>(null);

async function submit() {
  error.value = null;
  if (newPassword.value.length < MIN_LENGTH) {
    error.value = `Das neue Passwort muss mindestens ${MIN_LENGTH} Zeichen haben.`;
    return;
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'Die Wiederholung stimmt nicht mit dem neuen Passwort überein.';
    return;
  }
  isSaving.value = true;
  try {
    await authStore.changePassword(currentPassword.value, newPassword.value);
    emit('changed');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Passwort konnte nicht geändert werden';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <div class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50" @click.self="emit('close')">
    <form role="dialog" aria-modal="true" aria-label="Passwort ändern" class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6" @submit.prevent="submit">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <KeyRound class="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 class="text-xl font-semibold">Passwort ändern</h2>
          <p class="text-sm text-gray-600">Andere Geräte werden dabei abgemeldet.</p>
        </div>
      </div>

      <div class="space-y-4">
        <label class="block">
          <span class="text-sm font-medium text-gray-700">Aktuelles Passwort</span>
          <input v-model="currentPassword" type="password" autocomplete="current-password" required
            class="mt-1 w-full rounded-lg border px-3 py-2" />
        </label>
        <label class="block">
          <span class="text-sm font-medium text-gray-700">Neues Passwort (mind. {{ MIN_LENGTH }} Zeichen)</span>
          <input v-model="newPassword" type="password" autocomplete="new-password" required
            class="mt-1 w-full rounded-lg border px-3 py-2" />
        </label>
        <label class="block">
          <span class="text-sm font-medium text-gray-700">Neues Passwort wiederholen</span>
          <input v-model="confirmPassword" type="password" autocomplete="new-password" required
            class="mt-1 w-full rounded-lg border px-3 py-2" />
        </label>

        <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700" role="alert">
          {{ error }}
        </div>
      </div>

      <div class="flex justify-end gap-3 mt-6">
        <button type="button" class="px-4 py-2 border rounded-lg hover:bg-gray-50" @click="emit('close')">
          Abbrechen
        </button>
        <button type="submit" :disabled="isSaving"
          class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50">
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          Passwort ändern
        </button>
      </div>
    </form>
  </div>
</template>
