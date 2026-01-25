<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { X, Eye, EyeOff, Lock } from 'lucide-vue-next';

const props = defineProps<{
  visible: boolean;
  changePasswordFn: (currentPassword: string, newPassword: string) => Promise<void>;
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'success'): void;
}>();

const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const showCurrentPassword = ref(false);
const showNewPassword = ref(false);
const showConfirmPassword = ref(false);
const isLoading = ref(false);
const error = ref<string | null>(null);
const success = ref(false);

const isValid = computed(() => {
  return (
    currentPassword.value.length > 0 &&
    newPassword.value.length >= 8 &&
    newPassword.value === confirmPassword.value
  );
});

const passwordMismatch = computed(() => {
  return confirmPassword.value.length > 0 && newPassword.value !== confirmPassword.value;
});

const passwordTooShort = computed(() => {
  return newPassword.value.length > 0 && newPassword.value.length < 8;
});

// Reset form when modal closes
watch(() => props.visible, (newVal) => {
  if (!newVal) {
    currentPassword.value = '';
    newPassword.value = '';
    confirmPassword.value = '';
    showCurrentPassword.value = false;
    showNewPassword.value = false;
    showConfirmPassword.value = false;
    error.value = null;
    success.value = false;
  }
});

async function handleSubmit() {
  if (!isValid.value) return;

  isLoading.value = true;
  error.value = null;

  try {
    await props.changePasswordFn(currentPassword.value, newPassword.value);
    success.value = true;
    emit('success');
    
    // Close modal after short delay
    setTimeout(() => {
      emit('update:visible', false);
    }, 1500);
  } catch (e: any) {
    error.value = e.message || 'Passwort konnte nicht geändert werden';
  } finally {
    isLoading.value = false;
  }
}

function close() {
  emit('update:visible', false);
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
      >
        <!-- Backdrop -->
        <div
          class="absolute inset-0 bg-black/50"
          @click="close"
        />

        <!-- Modal -->
        <div class="relative bg-white rounded-lg shadow-xl w-full max-w-md">
          <!-- Header -->
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
            <div class="flex items-center gap-2">
              <Lock class="w-5 h-5 text-gray-500" />
              <h2 class="text-lg font-semibold text-gray-900">Passwort ändern</h2>
            </div>
            <button
              @click="close"
              class="p-1 rounded-md hover:bg-gray-100 text-gray-500 hover:text-gray-700"
            >
              <X class="w-5 h-5" />
            </button>
          </div>

          <!-- Body -->
          <form @submit.prevent="handleSubmit" class="p-6 space-y-4">
            <!-- Success message -->
            <div
              v-if="success"
              class="p-4 bg-green-50 border border-green-200 rounded-md"
            >
              <p class="text-sm text-green-700">
                Passwort wurde erfolgreich geändert.
              </p>
            </div>

            <!-- Error message -->
            <div
              v-if="error"
              class="p-4 bg-red-50 border border-red-200 rounded-md"
            >
              <p class="text-sm text-red-700">{{ error }}</p>
            </div>

            <template v-if="!success">
              <!-- Current password -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Aktuelles Passwort
                </label>
                <div class="relative">
                  <input
                    v-model="currentPassword"
                    :type="showCurrentPassword ? 'text' : 'password'"
                    class="w-full px-3 py-2 pr-10 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
                    placeholder="Aktuelles Passwort eingeben"
                    autocomplete="current-password"
                  />
                  <button
                    type="button"
                    @click="showCurrentPassword = !showCurrentPassword"
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600"
                  >
                    <EyeOff v-if="showCurrentPassword" class="w-4 h-4" />
                    <Eye v-else class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <!-- New password -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Neues Passwort
                </label>
                <div class="relative">
                  <input
                    v-model="newPassword"
                    :type="showNewPassword ? 'text' : 'password'"
                    class="w-full px-3 py-2 pr-10 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
                    :class="{ 'border-red-300': passwordTooShort }"
                    placeholder="Neues Passwort eingeben"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    @click="showNewPassword = !showNewPassword"
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600"
                  >
                    <EyeOff v-if="showNewPassword" class="w-4 h-4" />
                    <Eye v-else class="w-4 h-4" />
                  </button>
                </div>
                <p
                  v-if="passwordTooShort"
                  class="mt-1 text-xs text-red-600"
                >
                  Mindestens 8 Zeichen erforderlich
                </p>
              </div>

              <!-- Confirm password -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Passwort bestätigen
                </label>
                <div class="relative">
                  <input
                    v-model="confirmPassword"
                    :type="showConfirmPassword ? 'text' : 'password'"
                    class="w-full px-3 py-2 pr-10 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
                    :class="{ 'border-red-300': passwordMismatch }"
                    placeholder="Neues Passwort wiederholen"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    @click="showConfirmPassword = !showConfirmPassword"
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600"
                  >
                    <EyeOff v-if="showConfirmPassword" class="w-4 h-4" />
                    <Eye v-else class="w-4 h-4" />
                  </button>
                </div>
                <p
                  v-if="passwordMismatch"
                  class="mt-1 text-xs text-red-600"
                >
                  Passwörter stimmen nicht überein
                </p>
              </div>

              <!-- Submit button -->
              <div class="pt-2">
                <button
                  type="submit"
                  :disabled="!isValid || isLoading"
                  class="w-full px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <span v-if="isLoading">Wird geändert...</span>
                  <span v-else>Passwort ändern</span>
                </button>
              </div>
            </template>
          </form>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .relative,
.modal-leave-active .relative {
  transition: transform 0.2s ease;
}

.modal-enter-from .relative,
.modal-leave-to .relative {
  transform: scale(0.95);
}
</style>
