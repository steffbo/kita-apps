<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { api } from '@/api';
import type {
  FeeScheduleConfig,
  FeeScheduleRequest,
  FeeScheduleStatus,
  FeeScheduleVersion,
  FeeTableRow,
} from '@/api/types';
import { Loader2, Plus, Trash2, Save, X, Pencil } from 'lucide-vue-next';
import { formatCurrency, formatDate } from '@/utils/format';

const CARE_HOURS = [30, 35, 40, 45, 50, 55];

const versions = ref<FeeScheduleVersion[]>([]);
const selectedId = ref<string | null>(null);
const isLoading = ref(true);
const loadError = ref<string | null>(null);

// Draft for creating a new version or editing a planned one.
const draft = ref<FeeScheduleRequest | null>(null);
const editingId = ref<string | null>(null);
const isSaving = ref(false);
const saveError = ref<string | null>(null);

const selected = computed(() => versions.value.find((v) => v.id === selectedId.value) ?? null);
const isEditing = computed(() => draft.value !== null);
// The config shown: the draft while editing, otherwise the selected version.
const shownConfig = computed<FeeScheduleConfig | null>(() => draft.value?.config ?? selected.value?.config ?? null);

// Fee tables of the shown config; the Ü3 table is optional and reference only.
const feeTables = computed(() => {
  const config = shownConfig.value;
  if (!config) return [];
  const tables: { key: string; title: string; hint?: string; rows: FeeTableRow[] }[] = [
    { key: 'entlastungTable', title: 'Entlastungstabelle (U3, ohne Geschwisterermäßigung)', rows: config.entlastungTable },
    { key: 'satzungTable', title: 'Satzungstabelle (U3; letzte Zeile = Höchstsatz, Durchschnitt = Pflegefamilie)', rows: config.satzungTable },
  ];
  if (config.kindergartenTable?.length) {
    tables.push({
      key: 'kindergartenTable',
      title: 'Kindergartentabelle (Ü3, ab dem vollendeten 3. Lebensjahr)',
      hint: 'Nur zur Information: Kindergartenkinder sind nach dem Elternbeitragsentlastungsgesetz beitragsfrei, die Berechnung verwendet diese Tabelle nicht.',
      rows: config.kindergartenTable,
    });
  }
  return tables;
});

async function loadVersions(selectId?: string): Promise<void> {
  isLoading.value = true;
  loadError.value = null;
  try {
    versions.value = await api.getFeeSchedules();
    const active = versions.value.find((v) => v.status === 'active');
    selectedId.value = selectId ?? selectedId.value ?? active?.id ?? versions.value[versions.value.length - 1]?.id ?? null;
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Beitragsordnung konnte nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
}

function firstOfNextMonth(): string {
  const now = new Date();
  const next = new Date(now.getFullYear(), now.getMonth() + 1, 1);
  return `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}-01`;
}

function cloneConfig(config: FeeScheduleConfig): FeeScheduleConfig {
  return JSON.parse(JSON.stringify(config)) as FeeScheduleConfig;
}

function startCreate(): void {
  const base = versions.value[versions.value.length - 1];
  if (!base) return;
  draft.value = { validFrom: firstOfNextMonth(), name: '', config: cloneConfig(base.config) };
  editingId.value = null;
  saveError.value = null;
}

function startEdit(version: FeeScheduleVersion): void {
  draft.value = { validFrom: version.validFrom, name: version.name, config: cloneConfig(version.config) };
  editingId.value = version.id;
  selectedId.value = version.id;
  saveError.value = null;
}

function cancelEdit(): void {
  draft.value = null;
  editingId.value = null;
  saveError.value = null;
}

async function saveDraft(): Promise<void> {
  if (!draft.value) return;
  isSaving.value = true;
  saveError.value = null;
  try {
    const saved = editingId.value
      ? await api.updateFeeSchedule(editingId.value, draft.value)
      : await api.createFeeSchedule(draft.value);
    draft.value = null;
    editingId.value = null;
    await loadVersions(saved.id);
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen';
  } finally {
    isSaving.value = false;
  }
}

async function deleteVersion(version: FeeScheduleVersion): Promise<void> {
  if (!confirm(`Geplante Version „${version.name}“ ab ${formatDate(version.validFrom)} löschen?`)) return;
  try {
    await api.deleteFeeSchedule(version.id);
    selectedId.value = null;
    cancelEdit();
    await loadVersions();
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Löschen fehlgeschlagen';
  }
}

function addRow(table: FeeTableRow[]): void {
  const last = table[table.length - 1];
  table.push({ minIncome: last ? last.minIncome + 1000 : 0, rates: last ? [...last.rates] : [0, 0, 0, 0, 0, 0] });
}

function removeRow(table: FeeTableRow[], index: number): void {
  if (table.length > 1) table.splice(index, 1);
}

// Sibling factors are edited as discount percent: factor 0.9 = 10 % Ermäßigung.
function discountPercent(factor: number): number {
  return Math.round((1 - factor) * 10000) / 100;
}

function setDiscountPercent(index: number, value: string): void {
  if (!draft.value) return;
  const percent = Number(value);
  draft.value.config.siblingDiscountFactors[index] = Math.round((1 - percent / 100) * 10000) / 10000;
}

const statusLabel: Record<FeeScheduleStatus, string> = {
  active: 'Aktuell gültig',
  planned: 'Geplant',
  past: 'Abgelaufen',
};

const statusTone: Record<FeeScheduleStatus, string> = {
  active: 'bg-green-100 text-green-700',
  planned: 'bg-blue-100 text-blue-700',
  past: 'bg-gray-100 text-gray-600',
};

onMounted(() => loadVersions());
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Beitragsordnung</h1>
        <p class="text-gray-600 mt-1 max-w-3xl">
          Beitragstabellen, Einkommensgrenzen, Geschwisterermäßigung, Essensgeld und Mitgliedsbeitrag.
          Jede Version gilt ab ihrem Datum bis zum Beginn der nächsten. Versionen, die bereits gelten,
          sind schreibgeschützt. Änderungen werden als neue Version ab einem künftigen Monatsersten angelegt.
        </p>
      </div>
      <button
        v-if="!isEditing"
        @click="startCreate"
        :disabled="versions.length === 0"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
      >
        <Plus class="h-4 w-4" />
        Neue Version
      </button>
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <div v-else-if="loadError" class="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
      <p class="text-red-600">{{ loadError }}</p>
      <button @click="loadVersions()" class="mt-2 text-sm text-red-700 underline">Erneut versuchen</button>
    </div>

    <div v-if="!isLoading" class="grid gap-6 lg:grid-cols-[18rem_1fr]">
      <!-- Versions -->
      <div class="bg-white rounded-xl border divide-y h-fit">
        <button
          v-for="version in [...versions].reverse()"
          :key="version.id"
          @click="!isEditing && (selectedId = version.id)"
          :disabled="isEditing"
          :class="[
            'w-full text-left px-4 py-3 transition-colors disabled:cursor-not-allowed',
            selectedId === version.id && !(isEditing && !editingId) ? 'bg-primary/5' : 'hover:bg-gray-50',
          ]"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="font-medium text-gray-900 truncate">{{ version.name }}</span>
            <span class="px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap" :class="statusTone[version.status]">
              {{ statusLabel[version.status] }}
            </span>
          </div>
          <p class="text-sm text-gray-500">
            ab {{ formatDate(version.validFrom) }}<span v-if="version.validUntil"> bis {{ formatDate(version.validUntil) }}</span>
          </p>
        </button>
        <div v-if="isEditing && !editingId" class="px-4 py-3 bg-primary/5">
          <span class="font-medium text-gray-900">Neue Version (Entwurf)</span>
        </div>
      </div>

      <!-- Details / editor -->
      <div v-if="shownConfig" class="space-y-6">
        <div class="bg-white rounded-xl border p-6">
          <div v-if="draft" class="grid gap-4 sm:grid-cols-2">
            <label class="block">
              <span class="text-sm font-medium text-gray-700">Name</span>
              <input v-model="draft.name" type="text" placeholder="z. B. Elternbeitragsordnung 2027"
                class="mt-1 w-full rounded-lg border px-3 py-2" />
            </label>
            <label class="block">
              <span class="text-sm font-medium text-gray-700">Gültig ab (Monatserster, in der Zukunft)</span>
              <input v-model="draft.validFrom" type="date" class="mt-1 w-full rounded-lg border px-3 py-2" />
            </label>
          </div>
          <div v-else-if="selected" class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900">{{ selected.name }}</h2>
              <p class="text-sm text-gray-600">
                gültig ab {{ formatDate(selected.validFrom) }}<span v-if="selected.validUntil"> bis {{ formatDate(selected.validUntil) }}</span>
              </p>
            </div>
            <div v-if="selected.editable" class="flex gap-2">
              <button @click="startEdit(selected)" class="inline-flex items-center gap-1 px-3 py-1.5 border rounded-lg text-sm hover:bg-gray-50">
                <Pencil class="h-4 w-4" /> Bearbeiten
              </button>
              <button @click="deleteVersion(selected)" class="inline-flex items-center gap-1 px-3 py-1.5 border border-red-200 text-red-700 rounded-lg text-sm hover:bg-red-50">
                <Trash2 class="h-4 w-4" /> Löschen
              </button>
            </div>
            <p v-else class="text-sm text-gray-500">Schreibgeschützt, da bereits gültig.</p>
          </div>
        </div>

        <!-- General amounts -->
        <div class="bg-white rounded-xl border p-6">
          <h3 class="font-semibold text-gray-900 mb-4">Grenzen und feste Beträge</h3>
          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <label class="block">
              <span class="text-sm text-gray-600">Beitragsfrei bis Jahreseinkommen</span>
              <input v-if="draft" v-model.number="draft.config.freeIncomeLimit" type="number" step="0.01" min="0" class="mt-1 w-full rounded-lg border px-3 py-2" />
              <p v-else class="font-medium">{{ formatCurrency(shownConfig.freeIncomeLimit) }}</p>
            </label>
            <label class="block">
              <span class="text-sm text-gray-600">Entlastungstabelle bis Jahreseinkommen</span>
              <input v-if="draft" v-model.number="draft.config.entlastungIncomeLimit" type="number" step="0.01" min="0" class="mt-1 w-full rounded-lg border px-3 py-2" />
              <p v-else class="font-medium">{{ formatCurrency(shownConfig.entlastungIncomeLimit) }}</p>
            </label>
            <label class="block">
              <span class="text-sm text-gray-600">Beitragsfrei ab Kinderzahl</span>
              <input v-if="draft" v-model.number="draft.config.siblingsFreeThreshold" type="number" step="1" min="1" class="mt-1 w-full rounded-lg border px-3 py-2" />
              <p v-else class="font-medium">{{ shownConfig.siblingsFreeThreshold }} Kinder</p>
            </label>
            <label class="block">
              <span class="text-sm text-gray-600">Essensgeld pro Monat</span>
              <input v-if="draft" v-model.number="draft.config.monthlyFoodFee" type="number" step="0.01" min="0" class="mt-1 w-full rounded-lg border px-3 py-2" />
              <p v-else class="font-medium">{{ formatCurrency(shownConfig.monthlyFoodFee) }}</p>
            </label>
            <label class="block">
              <span class="text-sm text-gray-600">Mitgliedsbeitrag pro Jahr</span>
              <input v-if="draft" v-model.number="draft.config.annualMembershipFee" type="number" step="0.01" min="0" class="mt-1 w-full rounded-lg border px-3 py-2" />
              <p v-else class="font-medium">{{ formatCurrency(shownConfig.annualMembershipFee) }}</p>
            </label>
          </div>

          <h4 class="font-medium text-gray-900 mt-6 mb-2">Geschwisterermäßigung (Satzungstabelle und Höchstsatz)</h4>
          <div class="flex flex-wrap gap-3">
            <div v-for="(factor, index) in shownConfig.siblingDiscountFactors" :key="index" class="rounded-lg border px-3 py-2 text-sm">
              <div class="text-gray-600">{{ index + 1 }}{{ index + 1 === shownConfig.siblingDiscountFactors.length ? '+' : '' }} {{ index === 0 ? 'Kind' : 'Kinder' }}</div>
              <div v-if="draft" class="flex items-center gap-1">
                <input :value="discountPercent(factor)" @input="setDiscountPercent(index, ($event.target as HTMLInputElement).value)"
                  type="number" step="1" min="0" max="99" class="w-16 rounded border px-2 py-1" />
                <span>%</span>
                <button v-if="draft.config.siblingDiscountFactors.length > 1 && index === draft.config.siblingDiscountFactors.length - 1"
                  @click="draft.config.siblingDiscountFactors.pop()" class="text-gray-400 hover:text-red-600" title="Entfernen">
                  <X class="h-4 w-4" />
                </button>
              </div>
              <div v-else class="font-medium">{{ discountPercent(factor) }} % Ermäßigung</div>
            </div>
            <button v-if="draft" @click="draft.config.siblingDiscountFactors.push(draft.config.siblingDiscountFactors[draft.config.siblingDiscountFactors.length - 1] ?? 1)"
              class="rounded-lg border border-dashed px-3 py-2 text-sm text-gray-600 hover:bg-gray-50">
              <Plus class="h-4 w-4 inline" /> Stufe
            </button>
          </div>
        </div>

        <!-- Tables -->
        <div
          v-for="table in feeTables"
          :key="table.key"
          class="bg-white rounded-xl border overflow-hidden"
        >
          <h3 class="font-semibold text-gray-900 px-6 pt-5 pb-3">{{ table.title }}</h3>
          <p v-if="table.hint" class="px-6 pb-3 -mt-1 text-sm text-gray-500">{{ table.hint }}</p>
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-gray-50">
                <tr class="text-left text-gray-500">
                  <th class="px-4 py-2 font-medium">Einkommen ab</th>
                  <th v-for="hours in CARE_HOURS" :key="hours" class="px-4 py-2 font-medium text-right">{{ hours }} h</th>
                  <th v-if="draft" class="px-2 py-2"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, rowIndex) in table.rows" :key="rowIndex" class="border-t">
                  <td class="px-4 py-2">
                    <input v-if="draft" v-model.number="row.minIncome" type="number" step="0.01" min="0" class="w-32 rounded border px-2 py-1" />
                    <span v-else>{{ formatCurrency(row.minIncome) }}</span>
                  </td>
                  <td v-for="(_, rateIndex) in CARE_HOURS" :key="rateIndex" class="px-4 py-2 text-right">
                    <input v-if="draft" v-model.number="row.rates[rateIndex]" type="number" step="0.01" min="0" class="w-24 rounded border px-2 py-1 text-right" />
                    <span v-else>{{ formatCurrency(row.rates[rateIndex] ?? 0) }}</span>
                  </td>
                  <td v-if="draft" class="px-2 py-2">
                    <button @click="removeRow(table.rows, rowIndex)" :disabled="table.rows.length <= 1"
                      class="text-gray-400 hover:text-red-600 disabled:opacity-30" title="Zeile entfernen">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="draft" class="px-6 py-3 border-t">
            <button @click="addRow(table.rows)" class="text-sm text-primary hover:underline">
              <Plus class="h-4 w-4 inline" /> Zeile hinzufügen
            </button>
          </div>
        </div>

        <!-- Editor actions -->
        <div v-if="draft" class="bg-white rounded-xl border p-4 flex flex-wrap items-center gap-3 sticky bottom-4">
          <div v-if="saveError" class="w-full p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700" role="alert">
            {{ saveError }}
          </div>
          <button @click="saveDraft" :disabled="isSaving"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50">
            <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ editingId ? 'Änderungen speichern' : 'Version anlegen' }}
          </button>
          <button @click="cancelEdit" class="px-4 py-2 border rounded-lg hover:bg-gray-50">Abbrechen</button>
          <p class="text-sm text-gray-500">
            Gilt ab {{ draft.validFrom ? formatDate(draft.validFrom) : '–' }} für Beitragserzeugung, Einstufungen und den Rechner.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
