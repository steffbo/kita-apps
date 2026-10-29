<script setup lang="ts">
import PageHeader from '@/components/ui/PageHeader.vue';

import { ref, computed, onMounted } from 'vue';
import { api } from '@/api';
import type {
  FeeScheduleConfig,
  FeeScheduleRequest,
  FeeScheduleStatus,
  FeeScheduleVersion,
  FeeTableRow,
} from '@/api/types';
import { Info, Loader2, Plus, Trash2, Save, X, Pencil } from 'lucide-vue-next';
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
  active: 'badge-success',
  planned: 'badge-primary',
  past: '',
};

onMounted(() => loadVersions());
</script>

<template>
  <div>
    <PageHeader title="Beitragsordnung">
      <template #actions>
        <button
          type="button"
          v-if="!isEditing"
          @click="startCreate"
          :disabled="versions.length === 0"
          class="btn btn-primary">
          <Plus class="h-4 w-4" />
          Neue Version
        </button>
      </template>
    </PageHeader>
    <div class="note note-info mb-5">
      <Info />
      <div class="min-w-0 flex-1">
        Jede Version gilt ab ihrem Datum bis zum Beginn der nächsten. Versionen, die bereits gelten,
        sind schreibgeschützt. Änderungen werden als neue Version ab einem künftigen Monatsersten angelegt.
      </div>
    </div>
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>
    <div v-else-if="loadError" class="note note-danger mb-4">
      <Info />
      <div class="min-w-0 flex-1">
        <p class="text-destructive">{{ loadError }}</p>
        <button type="button" @click="loadVersions()" class="btn btn-secondary">Erneut versuchen</button>
      </div>
    </div>
    <div v-if="!isLoading" class="grid gap-6 lg:grid-cols-[18rem_minmax(0,1fr)]">
      <!-- Versions -->
      <div class="card divide-y h-fit">
        <button
          type="button"
          v-for="version in [...versions].reverse()"
          :key="version.id"
          @click="!isEditing && (selectedId = version.id)"
          :disabled="isEditing"
          :class="[
          'w-full text-left px-4 py-3 transition-colors disabled:cursor-not-allowed',
          selectedId === version.id && !(isEditing && !editingId) ? 'bg-primary-soft' : 'hover:bg-accent',
          ]">
          <div class="flex items-center justify-between gap-2">
            <span class="font-medium text-foreground truncate">{{ version.name }}</span>
            <span class="badge" :class="statusTone[version.status]">{{ statusLabel[version.status] }}</span>
          </div>
          <p class="text-sm text-muted-foreground">
            ab {{ formatDate(version.validFrom) }}
            <span v-if="version.validUntil">bis {{ formatDate(version.validUntil) }}</span>
          </p>
        </button>
        <div v-if="isEditing && !editingId" class="px-4 py-3 bg-primary-soft">
          <span class="font-medium text-foreground">Neue Version (Entwurf)</span>
        </div>
      </div>
      <!-- Details / editor -->
      <div v-if="shownConfig" class="min-w-0 space-y-6">
        <div class="card p-4 sm:p-6">
          <div v-if="draft" class="grid gap-4 sm:grid-cols-2">
            <label class="label">
              <span class="text-sm font-medium text-foreground">Name</span>
              <input
                v-model="draft.name"
                type="text"
                placeholder="z. B. Elternbeitragsordnung 2027"
                class="input mt-1" />
            </label>
            <label class="label">
              <span class="text-sm font-medium text-foreground">Gültig ab (Monatserster, in der Zukunft)</span>
              <input v-model="draft.validFrom" type="date" class="input mt-1" />
            </label>
          </div>
          <div v-else-if="selected" class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-foreground">{{ selected.name }}</h2>
              <p class="text-sm text-muted-foreground">
                gültig ab {{ formatDate(selected.validFrom) }}
                <span v-if="selected.validUntil">bis {{ formatDate(selected.validUntil) }}</span>
              </p>
            </div>
            <div v-if="selected.editable" class="flex gap-2">
              <button type="button" @click="startEdit(selected)" class="btn btn-secondary">
                <Pencil class="h-4 w-4" />
                Bearbeiten
              </button>
              <button type="button" @click="deleteVersion(selected)" class="btn btn-danger-outline">
                <Trash2 class="h-4 w-4" />
                Löschen
              </button>
            </div>
            <p v-else class="text-sm text-muted-foreground">Schreibgeschützt, da bereits gültig.</p>
          </div>
        </div>
        <!-- General amounts -->
        <div class="card p-4 sm:p-6">
          <h3 class="font-semibold text-foreground mb-4">Grenzen und feste Beträge</h3>
          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <label class="label">
              <span class="text-sm text-muted-foreground">Beitragsfrei bis Jahreseinkommen</span>
              <input
                v-if="draft"
                v-model.number="draft.config.freeIncomeLimit"
                type="number"
                step="0.01"
                min="0"
                class="input text-right tabular-nums mt-1" />
              <p v-else class="text-right font-medium tabular-nums">
                {{ formatCurrency(shownConfig.freeIncomeLimit) }}
              </p>
            </label>
            <label class="label">
              <span class="text-sm text-muted-foreground">Entlastungstabelle bis Jahreseinkommen</span>
              <input
                v-if="draft"
                v-model.number="draft.config.entlastungIncomeLimit"
                type="number"
                step="0.01"
                min="0"
                class="input text-right tabular-nums mt-1" />
              <p v-else class="text-right font-medium tabular-nums">
                {{ formatCurrency(shownConfig.entlastungIncomeLimit) }}
              </p>
            </label>
            <label class="label">
              <span class="text-sm text-muted-foreground">Beitragsfrei ab Kinderzahl</span>
              <input
                v-if="draft"
                v-model.number="draft.config.siblingsFreeThreshold"
                type="number"
                step="1"
                min="1"
                class="input text-right tabular-nums mt-1" />
              <p v-else class="font-medium">{{ shownConfig.siblingsFreeThreshold }} Kinder</p>
            </label>
            <label class="label">
              <span class="text-sm text-muted-foreground">Essensgeld pro Monat</span>
              <input
                v-if="draft"
                v-model.number="draft.config.monthlyFoodFee"
                type="number"
                step="0.01"
                min="0"
                class="input text-right tabular-nums mt-1" />
              <p v-else class="text-right font-medium tabular-nums">{{ formatCurrency(shownConfig.monthlyFoodFee) }}</p>
            </label>
            <label class="label">
              <span class="text-sm text-muted-foreground">Mitgliedsbeitrag pro Jahr</span>
              <input
                v-if="draft"
                v-model.number="draft.config.annualMembershipFee"
                type="number"
                step="0.01"
                min="0"
                class="input text-right tabular-nums mt-1" />
              <p v-else class="text-right font-medium tabular-nums">
                {{ formatCurrency(shownConfig.annualMembershipFee) }}
              </p>
            </label>
          </div>
          <h4 class="font-medium text-foreground mt-6 mb-2">Geschwisterermäßigung (Satzungstabelle und Höchstsatz)</h4>
          <div class="flex flex-wrap gap-3">
            <div
              v-for="(factor, index) in shownConfig.siblingDiscountFactors"
              :key="index"
              class="rounded-lg border px-3 py-2 text-sm">
              <div class="text-muted-foreground">
                {{ index + 1 }}{{ index + 1 === shownConfig.siblingDiscountFactors.length ? '+' : '' }} {{ index === 0 ? 'Kind' : 'Kinder' }}
              </div>
              <div v-if="draft" class="flex items-center gap-1">
                <input
                  :value="discountPercent(factor)"
                  @input="setDiscountPercent(index, ($event.target as HTMLInputElement).value)"
                  type="number"
                  step="1"
                  min="0"
                  max="99"
                  class="input text-right tabular-nums w-16" />
                <span>%</span>
                <button
                  type="button"
                  v-if="draft.config.siblingDiscountFactors.length > 1 && index === draft.config.siblingDiscountFactors.length - 1"
                  @click="draft.config.siblingDiscountFactors.pop()"
                  class="btn btn-danger-outline btn-icon btn-sm"
                  aria-label="Entfernen"
                  title="Entfernen">
                  <X class="h-4 w-4" />
                </button>
              </div>
              <div v-else class="font-medium">{{ discountPercent(factor) }} % Ermäßigung</div>
            </div>
            <button
              type="button"
              v-if="draft"
              @click="draft.config.siblingDiscountFactors.push(draft.config.siblingDiscountFactors[draft.config.siblingDiscountFactors.length - 1] ?? 1)"
              class="btn btn-secondary">
              <Plus class="h-4 w-4 inline" />
              Stufe
            </button>
          </div>
        </div>
        <!-- Tables -->
        <div v-for="table in feeTables" :key="table.key" class="card overflow-hidden">
          <h3 class="font-semibold text-foreground px-6 pt-5 pb-3">{{ table.title }}</h3>
          <p v-if="table.hint" class="px-6 pb-3 -mt-1 text-sm text-muted-foreground">{{ table.hint }}</p>
          <ul class="divide-y sm:hidden">
            <li v-for="(row, rowIndex) in table.rows" :key="rowIndex" class="space-y-3 px-4 py-3">
              <div class="flex items-center justify-between gap-3">
                <label v-if="draft" class="label mb-0 min-w-0">
                  Einkommen ab
                  <input
                    v-model.number="row.minIncome"
                    type="number"
                    step="0.01"
                    min="0"
                    class="input mt-1 text-right tabular-nums" />
                </label>
                <p v-else class="font-medium tabular-nums">Einkommen ab {{ formatCurrency(row.minIncome) }}</p>
                <button
                  v-if="draft"
                  type="button"
                  @click="removeRow(table.rows, rowIndex)"
                  :disabled="table.rows.length <= 1"
                  class="btn btn-danger-outline btn-icon btn-sm"
                  title="Zeile entfernen"
                  aria-label="Zeile entfernen">
                  <Trash2 />
                </button>
              </div>
              <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <div
                  v-for="(hours, rateIndex) in CARE_HOURS"
                  :key="hours"
                  class="flex items-center justify-between gap-2">
                  <dt class="text-muted-foreground">{{ hours }} h</dt>
                  <dd class="text-right tabular-nums">
                    <input
                      v-if="draft"
                      v-model.number="row.rates[rateIndex]"
                      type="number"
                      step="0.01"
                      min="0"
                      :aria-label="`${hours} h`"
                      class="input w-24 text-right tabular-nums" />
                    <span v-else>{{ formatCurrency(row.rates[rateIndex] ?? 0) }}</span>
                  </dd>
                </div>
              </dl>
            </li>
          </ul>
          <div class="hidden overflow-x-auto sm:block">
            <table class="table">
              <thead class="bg-muted">
                <tr class="text-left text-muted-foreground">
                  <th class="px-4 py-2 font-medium">Einkommen ab</th>
                  <th v-for="hours in CARE_HOURS" :key="hours" class="px-4 py-2 font-medium !text-right tabular-nums">
                    {{ hours }} h
                  </th>
                  <th v-if="draft" class="px-2 py-2"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, rowIndex) in table.rows" :key="rowIndex" class="border-t">
                  <td class="px-4 py-2">
                    <input
                      v-if="draft"
                      v-model.number="row.minIncome"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input text-right tabular-nums w-32" />
                    <span v-else>{{ formatCurrency(row.minIncome) }}</span>
                  </td>
                  <td v-for="(_, rateIndex) in CARE_HOURS" :key="rateIndex" class="px-4 py-2 text-right tabular-nums">
                    <input
                      v-if="draft"
                      v-model.number="row.rates[rateIndex]"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input w-24 text-right tabular-nums" />
                    <span v-else>{{ formatCurrency(row.rates[rateIndex] ?? 0) }}</span>
                  </td>
                  <td v-if="draft" class="px-2 py-2">
                    <button
                      type="button"
                      @click="removeRow(table.rows, rowIndex)"
                      :disabled="table.rows.length <= 1"
                      class="btn btn-danger-outline btn-icon btn-sm"
                      aria-label="Zeile entfernen"
                      title="Zeile entfernen">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="draft" class="px-6 py-3 border-t">
            <button type="button" @click="addRow(table.rows)" class="btn btn-secondary">
              <Plus class="h-4 w-4 inline" />
              Zeile hinzufügen
            </button>
          </div>
        </div>
        <!-- Editor actions -->
        <div v-if="draft" class="card p-4 flex flex-wrap items-center gap-3 sticky bottom-4">
          <div v-if="saveError" class="note note-danger w-full" role="alert">
            <Info />
            <div class="min-w-0 flex-1">{{ saveError }}</div>
          </div>
          <button type="button" @click="cancelEdit" class="btn btn-secondary">Abbrechen</button>
          <button type="button" @click="saveDraft" :disabled="isSaving" class="btn btn-primary">
            <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ editingId ? 'Änderungen speichern' : 'Version anlegen' }}
          </button>
          <p class="text-sm text-muted-foreground">
            Gilt ab {{ draft.validFrom ? formatDate(draft.validFrom) : '–' }} für Beitragserzeugung, Einstufungen und den Rechner.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
