<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Parent, UpdateParentRequest, IncomeStatus } from '@/api/types';
import {
  ArrowLeft,
  Edit,
  Trash2,
  Loader2,
  User,
  Calendar,
  MapPin,
  X,
  Check,
  Mail,
  Phone,
  Users,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const parent = ref<Parent | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateParentRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

const parentId = computed(() => route.params.id as string);

async function loadParent() {
  isLoading.value = true;
  error.value = null;
  try {
    parent.value = await api.getParent(parentId.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadParent);

// ESC key handler to close all modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) {
      showDeleteDialog.value = false;
    } else if (showEditDialog.value) {
      showEditDialog.value = false;
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatDateForInput(dateStr: string): string {
  return dateStr.split('T')[0];
}

function formatCurrency(amount: number | undefined): string {
  if (!amount) return '-';
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0,
  }).format(amount);
}

function getIncomeStatusLabel(status?: IncomeStatus): string {
  switch (status) {
    case 'PROVIDED':
      return 'Einkommen angegeben';
    case 'MAX_ACCEPTED':
      return 'Höchstsatz akzeptiert';
    case 'PENDING':
      return 'Dokumente ausstehend';
    case 'NOT_REQUIRED':
      return 'Nicht erforderlich (Kind >3J bei Eintritt)';
    case 'HISTORIC':
      return 'Historisch (Kind jetzt >3J)';
    case 'FOSTER_FAMILY':
      return 'Pflegefamilie (Durchschnittsbeitrag)';
    default:
      return 'Nicht festgelegt';
  }
}

const incomeStatusOptions: { value: IncomeStatus; label: string }[] = [
  { value: '', label: 'Nicht festgelegt' },
  { value: 'PROVIDED', label: 'Einkommen angegeben' },
  { value: 'MAX_ACCEPTED', label: 'Höchstsatz akzeptiert' },
  { value: 'PENDING', label: 'Dokumente ausstehend' },
  { value: 'NOT_REQUIRED', label: 'Nicht erforderlich (Kind >3J bei Eintritt)' },
  { value: 'HISTORIC', label: 'Historisch (Kind jetzt >3J)' },
  { value: 'FOSTER_FAMILY', label: 'Pflegefamilie (Durchschnittsbeitrag)' },
];

function openEditDialog() {
  if (!parent.value) return;
  editForm.value = {
    firstName: parent.value.firstName,
    lastName: parent.value.lastName,
    birthDate: parent.value.birthDate ? formatDateForInput(parent.value.birthDate) : undefined,
    email: parent.value.email,
    phone: parent.value.phone,
    street: parent.value.street,
    streetNo: parent.value.streetNo,
    postalCode: parent.value.postalCode,
    city: parent.value.city,
    annualHouseholdIncome: parent.value.annualHouseholdIncome,
    incomeStatus: parent.value.incomeStatus || '',
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!parent.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateParent(parentId.value, editForm.value);
    parent.value = updated;
    showEditDialog.value = false;
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
}

async function handleDelete() {
  isDeleting.value = true;
  try {
    await api.deleteParent(parentId.value);
    router.push('/eltern');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function goToChild(childId: string) {
  router.push(`/kinder/${childId}`);
}
</script>

<template>
  <div>
    <!-- Back button -->
    <button
      @click="router.push('/eltern')"
      class="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-6"
    >
      <ArrowLeft class="h-4 w-4" />
      Zurück zur Übersicht
    </button>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadParent" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Parent details -->
    <div v-else-if="parent">
      <!-- Header -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-8 w-8 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                {{ parent.firstName }} {{ parent.lastName }}
              </h1>
              <p class="text-gray-600">Elternteil</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="openEditDialog"
              class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-5 w-5" />
            </button>
            <button
              v-if="authStore.isAdmin"
              @click="showDeleteDialog = true"
              class="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
              title="Löschen"
            >
              <Trash2 class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- Info grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mt-6 pt-6 border-t">
          <div v-if="parent.birthDate" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Geburtsdatum</p>
              <p class="font-medium">{{ formatDate(parent.birthDate) }}</p>
            </div>
          </div>
          <div v-if="parent.email" class="flex items-start gap-3">
            <Mail class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">E-Mail</p>
              <a :href="`mailto:${parent.email}`" class="font-medium text-primary hover:underline">
                {{ parent.email }}
              </a>
            </div>
          </div>
          <div v-if="parent.phone" class="flex items-start gap-3">
            <Phone class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Telefon</p>
              <a :href="`tel:${parent.phone}`" class="font-medium text-primary hover:underline">
                {{ parent.phone }}
              </a>
            </div>
          </div>
          <div v-if="parent.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Adresse</p>
              <p class="font-medium">{{ parent.street }} {{ parent.streetNo }}</p>
              <p class="text-sm text-gray-500">{{ parent.postalCode }} {{ parent.city }}</p>
            </div>
          </div>
        </div>

        <!-- Income section -->
        <div class="mt-6 pt-6 border-t">
          <h3 class="text-sm font-medium text-gray-500 mb-4">Einkommensinformationen</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
            <div>
              <p class="text-sm text-gray-500">Einkommensstatus</p>
              <p class="font-medium">{{ getIncomeStatusLabel(parent.incomeStatus) }}</p>
            </div>
            <div v-if="parent.incomeStatus === 'PROVIDED' || parent.incomeStatus === 'HISTORIC'">
              <p class="text-sm text-gray-500">Jahreshaushaltseinkommen</p>
              <p class="font-medium">{{ formatCurrency(parent.annualHouseholdIncome) }}</p>
            </div>
          </div>
        </div>

        <!-- Children Section -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex items-center gap-2 mb-4">
            <Users class="h-4 w-4 text-gray-500" />
            <h3 class="text-sm font-medium text-gray-500">Verknüpfte Kinder</h3>
          </div>

          <div v-if="parent.children && parent.children.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button
              v-for="child in parent.children"
              :key="child.id"
              @click="goToChild(child.id)"
              class="bg-gray-50 rounded-lg border p-4 hover:border-primary/50 hover:bg-gray-100 transition-colors text-left"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                  <User class="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p class="font-medium text-gray-900">{{ child.firstName }} {{ child.lastName }}</p>
                  <p class="text-sm text-gray-500 font-mono">{{ child.memberNumber }}</p>
                </div>
              </div>
            </button>
          </div>

          <div v-else class="text-center py-6 bg-gray-50 rounded-lg border border-dashed">
            <Users class="h-8 w-8 text-gray-400 mx-auto mb-2" />
            <p class="text-gray-500 text-sm">Keine Kinder verknüpft</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Dialog -->
    <div
      v-if="showEditDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showEditDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Elternteil bearbeiten</h2>
          <button @click="showEditDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
              <input
                id="edit-firstName"
                v-model="editForm.firstName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
              <input
                id="edit-lastName"
                v-model="editForm.lastName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input
              id="edit-birthDate"
              v-model="editForm.birthDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="edit-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              id="edit-email"
              v-model="editForm.email"
              type="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="edit-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
            <input
              id="edit-phone"
              v-model="editForm.phone"
              type="tel"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
              <input
                id="edit-street"
                v-model="editForm.street"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
              <input
                id="edit-streetNo"
                v-model="editForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div class="col-span-2">
              <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <!-- Income section -->
          <div class="pt-4 border-t">
            <h3 class="text-sm font-medium text-gray-700 mb-3">Einkommensinformationen</h3>
            
            <div class="mb-4">
              <label for="edit-incomeStatus" class="block text-sm font-medium text-gray-700 mb-1">Einkommensstatus</label>
              <select
                id="edit-incomeStatus"
                v-model="editForm.incomeStatus"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              >
                <option v-for="option in incomeStatusOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>

            <div v-if="editForm.incomeStatus === 'PROVIDED' || editForm.incomeStatus === 'HISTORIC'">
              <label for="edit-income" class="block text-sm font-medium text-gray-700 mb-1">Jahreshaushaltseinkommen</label>
              <input
                id="edit-income"
                v-model.number="editForm.annualHouseholdIncome"
                type="number"
                min="0"
                step="any"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div v-if="editError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ editError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showEditDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isEditing"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation Dialog -->
    <div
      v-if="showDeleteDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeleteDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 rounded-lg">
            <Trash2 class="h-6 w-6 text-red-600" />
          </div>
          <h2 class="text-xl font-semibold">Elternteil löschen?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ parent?.firstName }} {{ parent?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen zu Kindern werden ebenfalls entfernt.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleDelete"
            :disabled="isDeleting"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
