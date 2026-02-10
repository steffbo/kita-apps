<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type {
  Einstufung,
  Child,
  Household,
  IncomeDetails,
  HouseholdIncomeCalculation,
  CalculateIncomeResponse,
  CreateEinstufungRequest,
  UpdateEinstufungRequest,
} from '@/api/types';
import {
  ArrowLeft,
  Calculator,
  Loader2,
  Save,
  Euro,
  Info,
  User,
  Home,
  Clock,
  FileText,
  CheckCircle,
} from 'lucide-vue-next';
import IncomeForm from '@/components/IncomeForm.vue';
import EinstufungPDF from '@/components/EinstufungPDF.vue';

const route = useRoute();
const router = useRouter();

const isNew = computed(() => route.params.id === 'neu');
const einstufungId = computed(() => (isNew.value ? null : route.params.id as string));

const isLoading = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const saveSuccess = ref(false);

// Loaded einstufung (edit mode)
const einstufung = ref<Einstufung | null>(null);

// Form data
const selectedChildId = ref('');
const selectedYear = ref(new Date().getFullYear());
const validFrom = ref(`${new Date().getFullYear()}-01-01`);
const careHoursPerWeek = ref(45);
const childrenCount = ref(1);
const highestRateVoluntary = ref(false);
const notes = ref('');

// Income form
function emptyIncome(): IncomeDetails {
  return {
    grossIncome: 0,
    otherIncome: 0,
    socialSecurityShare: 0,
    privateInsurance: 0,
    tax: 0,
    advertisingCosts: 1500,
    profit: 0,
    welfareExpense: 0,
    selfEmployedTax: 0,
    parentalBenefit: 0,
    maternityBenefit: 0,
    insurances: 0,
    maintenanceToPay: 0,
    maintenanceReceived: 0,
  };
}

const parent1Income = ref<IncomeDetails>(emptyIncome());
const parent2Income = ref<IncomeDetails>(emptyIncome());

// Income calculation (local, no API call needed)
function calcEmployeeNet(i: IncomeDetails): number {
  return i.grossIncome + i.otherIncome - i.socialSecurityShare - i.privateInsurance - i.tax - i.advertisingCosts;
}
function calcSelfEmployedNet(i: IncomeDetails): number {
  return i.profit - i.welfareExpense - i.selfEmployedTax;
}
function calcFeeRelevant(i: IncomeDetails): number {
  return calcEmployeeNet(i) + calcSelfEmployedNet(i) - i.insurances - i.maintenanceToPay + i.maintenanceReceived;
}
function calcNetIncome(i: IncomeDetails): number {
  return calcFeeRelevant(i) + i.parentalBenefit + i.maternityBenefit;
}
function round2(v: number): number {
  return Math.round(v * 100) / 100;
}

const incomePreview = computed<CalculateIncomeResponse | null>(() => {
  if (highestRateVoluntary.value) return null;
  const p1 = parent1Income.value;
  const p2 = parent2Income.value;
  return {
    parent1NetIncome: round2(calcNetIncome(p1)),
    parent2NetIncome: round2(calcNetIncome(p2)),
    parent1FeeRelevantIncome: round2(calcFeeRelevant(p1)),
    parent2FeeRelevantIncome: round2(calcFeeRelevant(p2)),
    householdFeeIncome: round2(calcFeeRelevant(p1) + calcFeeRelevant(p2)),
    householdFullIncome: round2(calcNetIncome(p1) + calcNetIncome(p2)),
  };
});

// Reference data
const children = ref<Child[]>([]);
const households = ref<Household[]>([]);

// Computed: selected child's household
const selectedChild = computed(() => children.value.find((c) => c.id === selectedChildId.value));
const selectedHousehold = computed(() => {
  if (!selectedChild.value?.householdId) return null;
  return households.value.find((h) => h.id === selectedChild.value!.householdId);
});
const childMissingHousehold = computed(() => !!selectedChildId.value && !!selectedChild.value && !selectedChild.value.householdId);

// Care hour options
const careHourOptions = [30, 35, 40, 45, 50, 55];

// Income field definitions for the form
const employeeFields = [
  { key: 'grossIncome', label: 'Bruttoeinkommen', sign: '+' },
  { key: 'otherIncome', label: 'Sonstige Einnahmen', sign: '+' },
  { key: 'socialSecurityShare', label: 'AN-Anteile Sozialversicherung', sign: '-' },
  { key: 'privateInsurance', label: 'Private KV/PV', sign: '-' },
  { key: 'tax', label: 'Lohnsteuer / Kirchensteuer / Solidaritätszuschlag', sign: '-' },
  { key: 'advertisingCosts', label: 'Werbungskosten-Pauschale', sign: '-' },
] as const;

const selfEmployedFields = [
  { key: 'profit', label: 'Gewinn (Gewerbebetrieb / selbst. Arbeit)', sign: '+' },
  { key: 'welfareExpense', label: 'Abgabe für persönliche Daseinsfürsorge', sign: '-' },
  { key: 'selfEmployedTax', label: 'Steuern (ESt, KiSt, SolZu)', sign: '-' },
] as const;

const benefitFields = [
  { key: 'parentalBenefit', label: 'Elterngeld', sign: '+', hint: 'Nicht beitragsrelevant' },
  { key: 'maternityBenefit', label: 'Mutterschaftsgeld', sign: '+', hint: 'Nicht beitragsrelevant' },
  { key: 'insurances', label: 'Versicherungen', sign: '-' },
] as const;

const maintenanceFields = [
  { key: 'maintenanceToPay', label: 'Unterhalt (zu zahlen)', sign: '-' },
  { key: 'maintenanceReceived', label: 'Unterhalt (erhalten)', sign: '+' },
] as const;

async function loadData() {
  isLoading.value = true;
  error.value = null;
  try {
    const [childRes, householdRes] = await Promise.all([
      api.getChildren({ limit: 2000 }),
      api.getHouseholds({ limit: 2000 }),
    ]);
    children.value = childRes.data;
    households.value = householdRes.data;

    if (einstufungId.value) {
      const e = await api.getEinstufung(einstufungId.value);
      einstufung.value = e;
      selectedChildId.value = e.childId;
      selectedYear.value = e.year;
      validFrom.value = e.validFrom.split('T')[0];
      careHoursPerWeek.value = e.careHoursPerWeek;
      childrenCount.value = e.childrenCount;
      highestRateVoluntary.value = e.highestRateVoluntary;
      notes.value = e.notes || '';
      parent1Income.value = { ...emptyIncome(), ...e.incomeCalculation.parent1 };
      parent2Income.value = { ...emptyIncome(), ...e.incomeCalculation.parent2 };
    } else if (route.query.childId) {
      // Pre-select child from query param
      const childId = route.query.childId as string;
      selectedChildId.value = childId;
      // Ensure the child is in the list (fetch individually if needed)
      if (!children.value.find(c => c.id === childId)) {
        try {
          const child = await api.getChild(childId);
          children.value.unshift(child);
        } catch {
          // Child not found, ignore
        }
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}



async function handleSubmit() {
  if (!selectedChildId.value) {
    error.value = 'Bitte ein Kind auswählen';
    return;
  }

  isSaving.value = true;
  error.value = null;
  saveSuccess.value = false;

  const incomeCalculation: HouseholdIncomeCalculation = {
    parent1: parent1Income.value,
    parent2: parent2Income.value,
  };

  try {
    if (isNew.value) {
      const created = await api.createEinstufung({
        childId: selectedChildId.value,
        year: selectedYear.value,
        validFrom: validFrom.value,
        incomeCalculation,
        highestRateVoluntary: highestRateVoluntary.value,
        careHoursPerWeek: careHoursPerWeek.value,
        childrenCount: childrenCount.value,
        notes: notes.value || undefined,
      });
      // Stay on page: show result + PDF immediately
      einstufung.value = created;
      router.replace(`/einstufungen/${created.id}`);
      saveSuccess.value = true;
      setTimeout(() => (saveSuccess.value = false), 3000);
    } else {
      const updated = await api.updateEinstufung(einstufungId.value!, {
        incomeCalculation,
        highestRateVoluntary: highestRateVoluntary.value,
        careHoursPerWeek: careHoursPerWeek.value,
        childrenCount: childrenCount.value,
        validFrom: validFrom.value,
        notes: notes.value || undefined,
      });
      einstufung.value = updated;
      saveSuccess.value = true;
      setTimeout(() => (saveSuccess.value = false), 3000);
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSaving.value = false;
  }
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(amount);
}

onMounted(loadData);
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center gap-4 mb-6">
      <button
        @click="router.push('/einstufungen')"
        class="p-2 rounded-lg hover:bg-gray-100 transition-colors"
      >
        <ArrowLeft class="h-5 w-5" />
      </button>
      <div class="flex-1">
        <h1 class="text-2xl font-bold text-gray-900">
          {{ isNew ? 'Neue Einstufung' : 'Einstufung bearbeiten' }}
        </h1>
        <p v-if="einstufung && einstufung.child" class="text-sm text-gray-500 mt-0.5">
          {{ einstufung.child.firstName }} {{ einstufung.child.lastName }} &middot; {{ einstufung.year }}
        </p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 text-primary animate-spin" />
    </div>

    <!-- Error -->
    <div v-if="error" class="bg-red-50 text-red-700 p-4 rounded-lg mb-4">
      {{ error }}
    </div>

    <!-- Success -->
    <div v-if="saveSuccess" class="bg-green-50 text-green-700 p-4 rounded-lg mb-4 flex items-center gap-2">
      <CheckCircle class="h-5 w-5" />
      Einstufung erfolgreich gespeichert.
    </div>

    <form v-if="!isLoading" @submit.prevent="handleSubmit" class="space-y-6">
      <!-- Child & Parameters Card -->
      <div class="bg-white rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <User class="h-5 w-5 text-gray-400" />
          Kind & Parameter
        </h2>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <!-- Child select -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Kind *</label>
            <select
              v-model="selectedChildId"
              :disabled="!isNew"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-gray-50 disabled:text-gray-500"
              required
            >
              <option value="">— Kind auswählen —</option>
              <option v-for="c in children" :key="c.id" :value="c.id">
                {{ c.firstName }} {{ c.lastName }} ({{ c.memberNumber }})
              </option>
            </select>
          </div>

          <!-- Year -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Jahr *</label>
            <select
              v-model="selectedYear"
              :disabled="!isNew"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-gray-50"
              required
            >
              <option v-for="y in [selectedYear - 1, selectedYear, selectedYear + 1]" :key="y" :value="y">
                {{ y }}
              </option>
            </select>
          </div>

          <!-- Valid from -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Gültig ab *</label>
            <input
              type="date"
              v-model="validFrom"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
              required
            />
          </div>

          <!-- Care hours -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Betreuungsstunden pro Woche *</label>
            <select
              v-model.number="careHoursPerWeek"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
              required
            >
              <option v-for="h in careHourOptions" :key="h" :value="h">{{ h }} Stunden</option>
            </select>
          </div>

          <!-- Children count -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Anzahl unterhaltspflichtiger Kinder *</label>
            <input
              type="number"
              v-model.number="childrenCount"
              min="1"
              max="10"
              class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
              required
            />
          </div>

          <!-- Highest rate voluntary -->
          <div class="flex items-center gap-2 self-end pb-2">
            <input
              type="checkbox"
              v-model="highestRateVoluntary"
              id="highestRateVoluntary"
              class="h-4 w-4 text-primary rounded focus:ring-primary/20"
            />
            <label for="highestRateVoluntary" class="text-sm text-gray-700">
              Freiwillige Anerkennung des Höchstsatzes
            </label>
          </div>
        </div>

        <!-- Household info -->
        <div v-if="selectedHousehold" class="mt-4 p-3 bg-gray-50 rounded-lg flex items-center gap-2 text-sm text-gray-600">
          <Home class="h-4 w-4 text-gray-400" />
          Haushalt: <strong>{{ selectedHousehold.name }}</strong>
          <span v-if="selectedHousehold.annualHouseholdIncome" class="ml-2">
            (bisheriges Einkommen: {{ formatCurrency(selectedHousehold.annualHouseholdIncome) }})
          </span>
        </div>
        <div v-else-if="childMissingHousehold" class="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start gap-2 text-sm text-amber-800">
          <Info class="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
          <div>
            <strong>Kein Haushalt zugeordnet.</strong>
            Bitte zuerst Eltern anlegen und dem Kind einen Haushalt zuweisen, bevor eine Einstufung erstellt werden kann.
            <router-link
              :to="`/kinder/${selectedChildId}`"
              class="underline font-medium hover:text-amber-900"
            >
              Zum Kind →
            </router-link>
          </div>
        </div>

        <!-- Notes -->
        <div class="mt-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">Bemerkungen</label>
          <textarea
            v-model="notes"
            rows="2"
            class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
            placeholder="Optionale Bemerkungen..."
          ></textarea>
        </div>
      </div>

      <!-- Income Calculation Card -->
      <div v-if="!highestRateVoluntary" class="bg-white rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-1 flex items-center gap-2">
          <Calculator class="h-5 w-5 text-gray-400" />
          Einkommensberechnung
        </h2>
        <p class="text-sm text-gray-500 mb-4">
          Festsetzung des Elternbeitrages – alle Beträge als Jahressummen in EUR
        </p>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <!-- Parent 1 -->
          <div>
            <h3 class="font-medium text-gray-800 mb-3 pb-2 border-b">
              Elternteil 1 (Mutter / Sorgeberechtigte/r)
            </h3>
            <IncomeForm v-model="parent1Income" />
          </div>

          <!-- Parent 2 -->
          <div>
            <h3 class="font-medium text-gray-800 mb-3 pb-2 border-b">
              Elternteil 2 (Vater / Sorgeberechtigte/r)
            </h3>
            <IncomeForm v-model="parent2Income" />
          </div>
        </div>

        <!-- Calculation preview -->
        <div v-if="incomePreview" class="mt-6 p-4 bg-blue-50 rounded-lg border border-blue-100">
          <h4 class="text-sm font-semibold text-blue-900 mb-3 flex items-center gap-2">
            <Euro class="h-4 w-4" />
            Einkommensvorschau
          </h4>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div>
              <span class="text-blue-600">Elternteil 1 (Netto)</span>
              <p class="font-semibold text-blue-900">{{ formatCurrency(incomePreview.parent1NetIncome) }}</p>
              <span class="text-xs text-blue-500">Beitragsrelevant: {{ formatCurrency(incomePreview.parent1FeeRelevantIncome) }}</span>
            </div>
            <div>
              <span class="text-blue-600">Elternteil 2 (Netto)</span>
              <p class="font-semibold text-blue-900">{{ formatCurrency(incomePreview.parent2NetIncome) }}</p>
              <span class="text-xs text-blue-500">Beitragsrelevant: {{ formatCurrency(incomePreview.parent2FeeRelevantIncome) }}</span>
            </div>
            <div>
              <span class="text-blue-600">Haushaltseinkommen (beitragsrelevant)</span>
              <p class="text-lg font-bold text-blue-900">{{ formatCurrency(incomePreview.householdFeeIncome) }}</p>
              <span class="text-xs text-blue-500">Gesamt inkl. Leistungen: {{ formatCurrency(incomePreview.householdFullIncome) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Highest rate notice -->
      <div v-if="highestRateVoluntary" class="bg-orange-50 border border-orange-200 rounded-lg p-4 flex items-start gap-3">
        <Info class="h-5 w-5 text-orange-500 mt-0.5 shrink-0" />
        <div class="text-sm text-orange-800">
          <strong>Freiwillige Anerkennung des Höchstsatzes</strong>
          <p class="mt-1">
            Die Einkommensberechnung entfällt, da der Höchstsatz freiwillig anerkannt wird.
            Der Beitrag wird direkt nach dem Höchstsatz der Satzungstabelle berechnet.
          </p>
        </div>
      </div>

      <!-- Result preview (edit mode) -->
      <div v-if="einstufung" class="bg-white rounded-lg border shadow-sm p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <FileText class="h-5 w-5 text-gray-400" />
          Ergebnis
        </h2>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <span class="text-sm text-gray-500">Platzgeld</span>
            <p class="text-xl font-bold text-gray-900">{{ formatCurrency(einstufung.monthlyChildcareFee) }}</p>
            <span class="text-xs text-gray-400">monatlich</span>
          </div>
          <div>
            <span class="text-sm text-gray-500">Essengeld</span>
            <p class="text-xl font-bold text-gray-900">{{ formatCurrency(einstufung.monthlyFoodFee) }}</p>
            <span class="text-xs text-gray-400">monatlich</span>
          </div>
          <div>
            <span class="text-sm text-gray-500">Vereinsbeitrag</span>
            <p class="text-xl font-bold text-gray-900">{{ formatCurrency(einstufung.annualMembershipFee) }}</p>
            <span class="text-xs text-gray-400">jährlich</span>
          </div>
          <div>
            <span class="text-sm text-gray-500">Regel</span>
            <p class="text-lg font-semibold text-gray-900">{{ einstufung.feeRule }}</p>
            <span v-if="einstufung.discountPercent > 0" class="text-xs text-green-600">
              {{ einstufung.discountPercent }}% Geschwisterrabatt
            </span>
          </div>
        </div>

        <!-- Monthly table -->
        <div v-if="einstufung.monthlyTable && einstufung.monthlyTable.length" class="mt-6">
          <h3 class="text-sm font-medium text-gray-700 mb-2">Monatsübersicht</h3>
          <table class="min-w-full divide-y divide-gray-200 text-sm">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Monat</th>
                <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Bereich</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 uppercase">Platzgeld</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 uppercase">Essengeld</th>
                <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 uppercase">Vereinsbeitrag</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              <tr v-for="row in einstufung.monthlyTable" :key="`${row.year}-${row.month}`">
                <td class="px-3 py-2 text-gray-700">
                  {{ new Date(row.year, row.month - 1).toLocaleString('de-DE', { month: 'long', year: 'numeric' }) }}
                </td>
                <td class="px-3 py-2 text-gray-600">{{ row.careType }} · {{ row.careHoursPerWeek }}h</td>
                <td class="px-3 py-2 text-right font-medium">{{ formatCurrency(row.childcareFee) }}</td>
                <td class="px-3 py-2 text-right">{{ formatCurrency(row.foodFee) }}</td>
                <td class="px-3 py-2 text-right">{{ row.membershipFee > 0 ? formatCurrency(row.membershipFee) : '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Submit -->
      <div class="flex items-center justify-between">
        <div>
          <EinstufungPDF v-if="einstufung" :einstufung="einstufung" />
        </div>
        <div class="flex items-center gap-3">
        <button
          type="button"
          @click="router.push('/einstufungen')"
          class="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
        >
          Abbrechen
        </button>
        <button
          type="submit"
          :disabled="isSaving || childMissingHousehold"
          class="inline-flex items-center gap-2 px-6 py-2 text-sm text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ isNew ? 'Einstufung erstellen' : 'Speichern' }}
        </button>
        </div>
      </div>
    </form>
  </div>
</template>
