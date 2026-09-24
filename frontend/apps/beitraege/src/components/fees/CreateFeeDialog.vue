<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Child, CreateFeeRequest } from '@/api/types';
import { Loader2, Plus, Search, User } from 'lucide-vue-next';
import { MONTH_OPTIONS as months } from '@/utils/format';

const emit = defineEmits<{ close: []; created: [] }>();

const currentYear = new Date().getFullYear();
const years = [currentYear - 1, currentYear, currentYear + 1];

const createFeeForm = ref<CreateFeeRequest>({
  childId: '',
  feeType: 'FOOD',
  year: currentYear,
  month: new Date().getMonth() + 1,
});
const isCreatingFee = ref(false);
const createFeeError = ref<string | null>(null);
const childSearchQuery = ref('');
const childSearchResults = ref<Child[]>([]);
const selectedChild = ref<Child | null>(null);
const isSearchingChildren = ref(false);

let childSearchTimeout: ReturnType<typeof setTimeout>;
let childSearchSeq = 0;
watch(childSearchQuery, (newVal) => {
  clearTimeout(childSearchTimeout);
  if (newVal.length < 2) {
    childSearchResults.value = [];
    return;
  }
  const seq = ++childSearchSeq;
  childSearchTimeout = setTimeout(async () => {
    isSearchingChildren.value = true;
    try {
      const response = await api.getChildren({ search: newVal, activeOnly: true, perPage: 10 });
      if (seq !== childSearchSeq) return; // a newer keystroke superseded this request
      childSearchResults.value = response.data;
    } catch {
      if (seq === childSearchSeq) childSearchResults.value = [];
    } finally {
      if (seq === childSearchSeq) isSearchingChildren.value = false;
    }
  }, 300);
});

onUnmounted(() => clearTimeout(childSearchTimeout));

function selectChild(child: Child) {
  selectedChild.value = child;
  createFeeForm.value.childId = child.id;
  childSearchQuery.value = '';
  childSearchResults.value = [];
}

function clearSelectedChild() {
  selectedChild.value = null;
  createFeeForm.value.childId = '';
}

const feeTypeRequiresMonth = computed(() => {
  return createFeeForm.value.feeType !== 'MEMBERSHIP';
});

async function handleCreateFee() {
  if (!selectedChild.value) {
    createFeeError.value = 'Bitte wählen Sie ein Kind aus';
    return;
  }

  isCreatingFee.value = true;
  createFeeError.value = null;

  try {
    const requestData: CreateFeeRequest = {
      childId: createFeeForm.value.childId,
      feeType: createFeeForm.value.feeType,
      year: createFeeForm.value.year,
    };

    // Only include month for non-membership fees
    if (feeTypeRequiresMonth.value && createFeeForm.value.month) {
      requestData.month = createFeeForm.value.month;
    }

    // Include optional fields if provided
    if (createFeeForm.value.amount) {
      requestData.amount = createFeeForm.value.amount;
    }
    if (createFeeForm.value.dueDate) {
      requestData.dueDate = createFeeForm.value.dueDate;
    }

    await api.createFee(requestData);
    emit('created');
  } catch (e) {
    createFeeError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen des Beitrags';
  } finally {
    isCreatingFee.value = false;
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <Plus class="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 class="text-xl font-semibold">Einzelnen Beitrag erstellen</h2>
          <p class="text-sm text-gray-600">Erstellt einen Beitrag für ein bestimmtes Kind</p>
        </div>
      </div>

      <div class="space-y-4">
        <!-- Child Selection -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Kind *</label>

          <!-- Selected Child Display -->
          <div v-if="selectedChild" class="flex items-center justify-between p-3 bg-gray-50 border border-gray-200 rounded-lg">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-primary/10 rounded-full">
                <User class="h-4 w-4 text-primary" />
              </div>
              <div>
                <p class="font-medium">{{ selectedChild.firstName }} {{ selectedChild.lastName }}</p>
                <p class="text-sm text-gray-500">Mitgl.-Nr.: {{ selectedChild.memberNumber }}</p>
              </div>
            </div>
            <button
              @click="clearSelectedChild"
              class="text-gray-400 hover:text-gray-600"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Child Search -->
          <div v-else class="relative">
            <div class="relative">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input
                v-model="childSearchQuery"
                type="text"
                placeholder="Name oder Mitgliedsnummer eingeben..."
                class="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
              <Loader2 v-if="isSearchingChildren" class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-gray-400" />
            </div>

            <!-- Search Results Dropdown -->
            <div
              v-if="childSearchResults.length > 0"
              class="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-auto"
            >
              <button
                v-for="child in childSearchResults"
                :key="child.id"
                @click="selectChild(child)"
                class="w-full flex items-center gap-3 px-4 py-3 hover:bg-gray-50 text-left border-b last:border-b-0"
              >
                <div class="p-1.5 bg-gray-100 rounded-full">
                  <User class="h-4 w-4 text-gray-600" />
                </div>
                <div>
                  <p class="font-medium">{{ child.firstName }} {{ child.lastName }}</p>
                  <p class="text-sm text-gray-500">{{ child.memberNumber }}</p>
                </div>
              </button>
            </div>
          </div>
        </div>

        <!-- Fee Type -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Beitragsart *</label>
          <select
            v-model="createFeeForm.feeType"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option value="FOOD">Essensgeld</option>
            <option value="CHILDCARE">Platzgeld</option>
            <option value="MEMBERSHIP">Vereinsbeitrag</option>
            <option value="REMINDER">Mahngebühr</option>
          </select>
        </div>

        <!-- Year -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Jahr *</label>
          <select
            v-model="createFeeForm.year"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>

        <!-- Month (conditional) -->
        <div v-if="feeTypeRequiresMonth">
          <label class="block text-sm font-medium text-gray-700 mb-1">Monat *</label>
          <select
            v-model="createFeeForm.month"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option v-for="month in months" :key="month.value" :value="month.value">
              {{ month.label }}
            </option>
          </select>
        </div>

        <!-- Amount (optional) -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            Betrag (optional)
            <span class="font-normal text-gray-500">- wird automatisch berechnet wenn leer</span>
          </label>
          <div class="relative">
            <input
              v-model.number="createFeeForm.amount"
              type="number"
              step="0.01"
              min="0"
              placeholder="z.B. 45.40"
              class="w-full px-3 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">EUR</span>
          </div>
        </div>

        <!-- Due Date (optional) -->
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            Fälligkeitsdatum (optional)
            <span class="font-normal text-gray-500">- wird automatisch gesetzt wenn leer</span>
          </label>
          <input
            v-model="createFeeForm.dueDate"
            type="date"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <!-- Error Display -->
        <div v-if="createFeeError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ createFeeError }}</p>
        </div>

        <!-- Actions -->
        <div class="flex justify-end gap-3 pt-4">
          <button
            @click="$emit('close')"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleCreateFee"
            :disabled="isCreatingFee || !selectedChild"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingFee" class="h-4 w-4 animate-spin" />
            <Plus v-else class="h-4 w-4" />
            Beitrag erstellen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
