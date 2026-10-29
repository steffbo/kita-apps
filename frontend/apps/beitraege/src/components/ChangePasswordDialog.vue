<script setup lang="ts">
import { ref } from 'vue';
import { KeyRound, Loader2 } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import AppSheet from '@/components/ui/AppSheet.vue';

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
  <AppSheet title="Passwort ändern" size="sm" @close="emit('close')">
    <form id="change-password" class="space-y-4" @submit.prevent="submit">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <KeyRound class="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 class="text-xl font-semibold">Passwort ändern</h2>
          <p class="text-sm text-muted-foreground">Andere Geräte werden dabei abgemeldet.</p>
        </div>
      </div>

      <div class="space-y-4">
        <label class="block">
          <span class="text-sm font-medium text-foreground">Aktuelles Passwort</span>
          <input v-model="currentPassword" type="password" autocomplete="current-password" required
            class="mt-1 w-full input" />
        </label>
        <label class="block">
          <span class="text-sm font-medium text-foreground">Neues Passwort (mind. {{ MIN_LENGTH }} Zeichen)</span>
          <input v-model="newPassword" type="password" autocomplete="new-password" required
            class="mt-1 w-full input" />
        </label>
        <label class="block">
          <span class="text-sm font-medium text-foreground">Neues Passwort wiederholen</span>
          <input v-model="confirmPassword" type="password" autocomplete="new-password" required
            class="mt-1 w-full input" />
        </label>

        <div v-if="error" class="note note-danger" role="alert">
          {{ error }}
        </div>
      </div>

    </form>
    <template #footer>
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          Abbrechen
        </button>
        <button type="submit" form="change-password" :disabled="isSaving" class="btn btn-primary">
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          Passwort ändern
        </button>
    </template>
  </AppSheet>
</template>
