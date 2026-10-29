<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';

import { ref, computed, watch, onUnmounted } from 'vue';
import { api } from '@/api';
import type { Child, CreateFeeRequest } from '@/api/types';
import { Info, Loader2, Plus, User } from 'lucide-vue-next';
import { MONTH_OPTIONS as months } from '@/utils/format';
import SearchInput from '@/components/SearchInput.vue';

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
  <AppSheet title="Einzelnen Beitrag erstellen" @close="$emit('close')">
    <form id="create-fee" class="space-y-4" @submit.prevent="handleCreateFee">
      <!-- Child Selection -->
      <div>
        <label class="label">Kind *</label>
        <!-- Selected Child Display -->
        <div
          v-if="selectedChild"
          class="flex items-center justify-between p-3 bg-muted border border-border rounded-lg">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-primary-soft rounded-full">
              <User class="h-4 w-4 text-primary" />
            </div>
            <div>
              <p class="font-medium">{{ selectedChild.firstName }} {{ selectedChild.lastName }}</p>
              <p class="text-sm text-muted-foreground">Mitgl.-Nr.: {{ selectedChild.memberNumber }}</p>
            </div>
          </div>
          <button
            type="button"
            aria-label="Auswahl aufheben"
            @click="clearSelectedChild"
            class="btn btn-ghost btn-icon">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <!-- Child Search -->
        <div v-else class="relative">
          <SearchInput
            v-model="childSearchQuery"
            placeholder="Name oder Mitgliedsnummer eingeben..."
            :loading="isSearchingChildren" />
          <!-- Search Results Dropdown -->
          <div
            v-if="childSearchResults.length > 0"
            class="absolute z-10 w-full mt-1 bg-popover border border-border rounded-lg shadow-lg max-h-60
              overflow-auto">
            <button
              type="button"
              v-for="child in childSearchResults"
              :key="child.id"
              @click="selectChild(child)"
              class="w-full flex items-center gap-3 px-4 py-3 hover:bg-accent text-left border-b last:border-b-0">
              <div class="p-1.5 bg-muted rounded-full">
                <User class="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p class="font-medium">{{ child.firstName }} {{ child.lastName }}</p>
                <p class="text-sm text-muted-foreground">{{ child.memberNumber }}</p>
              </div>
            </button>
          </div>
        </div>
      </div>
      <!-- Fee Type -->
      <div>
        <label class="label">Beitragsart *</label>
        <select v-model="createFeeForm.feeType" class="input">
          <option value="FOOD">Essensgeld</option>
          <option value="CHILDCARE">Platzgeld</option>
          <option value="MEMBERSHIP">Vereinsbeitrag</option>
          <option value="REMINDER">Mahngebühr</option>
        </select>
      </div>
      <!-- Year -->
      <div>
        <label class="label">Jahr *</label>
        <select v-model="createFeeForm.year" class="input">
          <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
        </select>
      </div>
      <!-- Month (conditional) -->
      <div v-if="feeTypeRequiresMonth">
        <label class="label">Monat *</label>
        <select v-model="createFeeForm.month" class="input">
          <option v-for="month in months" :key="month.value" :value="month.value">{{ month.label }}</option>
        </select>
      </div>
      <!-- Amount (optional) -->
      <div>
        <label class="label">
          Betrag (optional)
          <span class="font-normal text-muted-foreground">- wird automatisch berechnet wenn leer</span>
        </label>
        <div class="relative">
          <input
            v-model.number="createFeeForm.amount"
            type="number"
            step="0.01"
            min="0"
            placeholder="z.B. 45.40"
            class="input pr-10 text-right tabular-nums" />
          <span class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground">EUR</span>
        </div>
      </div>
      <!-- Due Date (optional) -->
      <div>
        <label class="label">
          Fälligkeitsdatum (optional)
          <span class="font-normal text-muted-foreground">- wird automatisch gesetzt wenn leer</span>
        </label>
        <input v-model="createFeeForm.dueDate" type="date" class="input" />
      </div>
      <!-- Error Display -->
      <div v-if="createFeeError" class="note note-danger">
        <Info />
        <div class="min-w-0 flex-1">
          <p class="text-sm text-destructive">{{ createFeeError }}</p>
        </div>
      </div>
    </form>
    <template #footer>
      <button type="button" @click="$emit('close')" class="btn btn-secondary">Abbrechen</button>
      <button type="submit" form="create-fee" :disabled="isCreatingFee || !selectedChild" class="btn btn-primary">
        <Loader2 v-if="isCreatingFee" class="h-4 w-4 animate-spin" />
        <Plus v-else class="h-4 w-4" />
        Beitrag erstellen
      </button>
    </template>
  </AppSheet>
</template>
