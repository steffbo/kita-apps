<script setup lang="ts">
import { computed, ref } from 'vue';
import { Loader2, UserPlus, UserCog } from 'lucide-vue-next';
import { api } from '@/api';
import type { UserAccount, UserRole } from '@/api/types';

const MIN_LENGTH = 8;

const props = defineProps<{
  /** Account to edit; absent when creating a new one. */
  user?: UserAccount | null;
  /** True when editing the signed-in account (role and active flag are locked). */
  isSelf?: boolean;
}>();
const emit = defineEmits<{ close: []; saved: [] }>();

const isNew = computed(() => !props.user);
const email = ref(props.user?.email ?? '');
const firstName = ref(props.user?.firstName ?? '');
const lastName = ref(props.user?.lastName ?? '');
const role = ref<UserRole>(props.user?.role ?? 'USER');
const isActive = ref(props.user?.isActive ?? true);
const password = ref('');
const isSaving = ref(false);
const error = ref<string | null>(null);

async function submit() {
  error.value = null;
  if (isNew.value && password.value.length < MIN_LENGTH) {
    error.value = `Das Passwort muss mindestens ${MIN_LENGTH} Zeichen haben.`;
    return;
  }
  const data = {
    email: email.value.trim(),
    firstName: firstName.value.trim() || undefined,
    lastName: lastName.value.trim() || undefined,
    role: role.value,
    isActive: isActive.value,
  };
  isSaving.value = true;
  try {
    if (props.user) {
      await api.updateUser(props.user.id, data);
    } else {
      await api.createUser({ ...data, password: password.value });
    }
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="emit('close')">
    <form role="dialog" aria-modal="true" :aria-label="isNew ? 'Benutzer anlegen' : 'Benutzer bearbeiten'" class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6" @submit.prevent="submit">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <UserPlus v-if="isNew" class="h-6 w-6 text-primary" />
          <UserCog v-else class="h-6 w-6 text-primary" />
        </div>
        <h2 class="text-xl font-semibold">{{ isNew ? 'Benutzer anlegen' : 'Benutzer bearbeiten' }}</h2>
      </div>

      <div class="space-y-4">
        <label class="block">
          <span class="text-sm font-medium text-gray-700">E-Mail (Anmeldename) *</span>
          <input v-model="email" type="email" required autocomplete="off" class="mt-1 w-full rounded-lg border px-3 py-2" />
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="text-sm font-medium text-gray-700">Vorname</span>
            <input v-model="firstName" type="text" class="mt-1 w-full rounded-lg border px-3 py-2" />
          </label>
          <label class="block">
            <span class="text-sm font-medium text-gray-700">Nachname</span>
            <input v-model="lastName" type="text" class="mt-1 w-full rounded-lg border px-3 py-2" />
          </label>
        </div>
        <label class="block">
          <span class="text-sm font-medium text-gray-700">Rolle</span>
          <select v-model="role" :disabled="isSelf" class="mt-1 w-full rounded-lg border px-3 py-2 disabled:bg-gray-100">
            <option value="USER">Benutzer</option>
            <option value="ADMIN">Administrator</option>
          </select>
          <span class="mt-1 block text-xs text-gray-500">
            Administratoren dürfen zusätzlich Benutzer, Beitragsordnung, Erinnerungen und Bankabruf verwalten.
          </span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="isActive" type="checkbox" :disabled="isSelf" class="rounded" />
          <span class="text-sm text-gray-700">Aktiv (darf sich anmelden)</span>
        </label>
        <p v-if="isSelf" class="text-xs text-gray-500">
          Das eigene Konto kann nicht deaktiviert oder herabgestuft werden.
        </p>
        <label v-if="isNew" class="block">
          <span class="text-sm font-medium text-gray-700">Startpasswort (mind. {{ MIN_LENGTH }} Zeichen) *</span>
          <input v-model="password" type="text" required autocomplete="off" class="mt-1 w-full rounded-lg border px-3 py-2 font-mono" />
          <span class="mt-1 block text-xs text-gray-500">
            Bitte der Person mitteilen; sie kann es nach der Anmeldung selbst ändern.
          </span>
        </label>

        <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700" role="alert">
          {{ error }}
        </div>
      </div>

      <div class="flex justify-end gap-3 mt-6">
        <button type="button" class="px-4 py-2 border rounded-lg hover:bg-gray-50" @click="emit('close')">Abbrechen</button>
        <button type="submit" :disabled="isSaving"
          class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50">
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          {{ isNew ? 'Anlegen' : 'Speichern' }}
        </button>
      </div>
    </form>
  </div>
</template>
