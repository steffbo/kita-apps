<script setup lang="ts">
import { ArrowLeft, Upload, Loader2, Check, AlertTriangle, AlertCircle, XCircle, Plus, Pencil, X, GitMerge, Link, RefreshCw } from 'lucide-vue-next';
import { useChildImportWizardContext } from '@/composables/useChildImportWizard';

const {
  isLoading,
  previewResult,
  selectedRows,
  mergeRows,
  editingRow,
  editedData,
  selectedValidCount,
  mergeRowsCount,
  sortedPreviewRows,
  toggleRow,
  toggleMerge,
  selectAll,
  deselectAll,
  setParentDecision,
  getParentDecision,
  setConflictResolution,
  getConflictResolution,
  rowHasConflicts,
  startEditing,
  cancelEditing,
  saveEditing,
  executeImport,
  goBack,
} = useChildImportWizardContext();
</script>

<template>
  <div class="space-y-6">
    <!-- Summary -->
    <div class="bg-card rounded-xl border p-6">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold">Vorschau</h2>
          <p class="text-sm text-muted-foreground mt-1">
            {{ previewResult?.validCount || 0 }} gültige Einträge,
            {{ previewResult?.errorCount || 0 }} mit Fehlern/Duplikaten
          </p>
        </div>
        <div class="flex items-center gap-4">
          <button @click="selectAll" class="text-sm text-primary hover:underline">
            Alle auswählen
          </button>
          <button @click="deselectAll" class="text-sm text-muted-foreground hover:underline">
            Alle abwählen
          </button>
          <div class="text-sm font-medium text-foreground">
            {{ selectedValidCount - mergeRowsCount }} neu,
            <span v-if="mergeRowsCount > 0" class="text-blue-600 dark:text-blue-300">{{ mergeRowsCount }} Merge</span>
            <span v-else class="text-muted-foreground">0 Merge</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Preview table -->
    <div class="bg-card rounded-xl border overflow-hidden">
      <div class="overflow-x-auto max-h-[500px] overflow-y-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted sticky top-0">
            <tr>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground w-12"></th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Mitglieds-Nr.</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Geburtsdatum</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Eintrittsdatum</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Rechtsanspr.</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Betreuung</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Elternteil 1</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground">Elternteil 2</th>
              <th class="px-4 py-3 text-left font-medium text-muted-foreground w-20"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <template
              v-for="row in sortedPreviewRows"
              :key="row.index"
            >
            <tr
              :class="[
                'hover:bg-accent',
                // Invalid rows: red
                !row.isValid && !row.isDuplicate ? 'bg-red-50 dark:bg-red-950/40' : '',
                // Duplicates not marked for merge: amber/yellow
                row.isDuplicate && !mergeRows.has(row.index) ? 'bg-amber-50 dark:bg-amber-950/40' : '',
                // Duplicates marked for merge: blue
                row.isDuplicate && mergeRows.has(row.index) ? 'bg-blue-50 dark:bg-blue-950/40' : '',
                // Selected valid rows: primary
                selectedRows.has(row.index) && row.isValid && !row.isDuplicate ? 'bg-primary/5' : '',
              ]"
            >
              <!-- Checkbox / Merge toggle -->
              <td class="px-4 py-3">
                <!-- For valid non-duplicates: normal checkbox -->
                <input
                  v-if="row.isValid && !row.isDuplicate"
                  type="checkbox"
                  :checked="selectedRows.has(row.index)"
                  @change="toggleRow(row.index)"
                  class="h-4 w-4 text-primary rounded border-border focus:ring-primary"
                />
                <!-- For duplicates with existing child: merge toggle -->
                <button
                  v-else-if="row.isDuplicate && row.existingChildId"
                  @click="toggleMerge(row.index)"
                  :class="[
                    'flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors',
                    mergeRows.has(row.index)
                      ? 'bg-blue-600 text-white'
                      : 'bg-muted text-foreground hover:bg-accent',
                  ]"
                  :title="mergeRows.has(row.index) ? 'Zusammenführung deaktivieren' : 'Eltern zu bestehendem Kind hinzufügen'"
                >
                  <GitMerge class="h-3 w-3" />
                  {{ mergeRows.has(row.index) ? 'Merge' : 'Merge?' }}
                </button>
                <!-- For invalid rows: disabled indicator -->
                <span v-else class="text-muted-foreground text-xs">-</span>
              </td>

              <!-- Status -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <!-- Action badge -->
                  <span 
                    v-if="row.action === 'create' && row.isValid && !row.isDuplicate"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-green-100 dark:bg-green-950/40 text-green-700 dark:text-green-300"
                  >
                    <Plus class="h-3 w-3" />
                    NEU
                  </span>
                  <span 
                    v-else-if="row.isDuplicate && mergeRows.has(row.index) && rowHasConflicts(row)"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-blue-100 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300"
                  >
                    <RefreshCw class="h-3 w-3" />
                    UPDATE
                  </span>
                  <span 
                    v-else-if="row.isDuplicate && mergeRows.has(row.index)"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-blue-100 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300"
                  >
                    <GitMerge class="h-3 w-3" />
                    MERGE
                  </span>
                  <span 
                    v-else-if="row.isDuplicate"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300"
                  >
                    EXISTIERT
                  </span>
                  <span 
                    v-else-if="!row.isValid"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-red-100 dark:bg-red-950/40 text-red-700 dark:text-red-300"
                  >
                    <XCircle class="h-3 w-3" />
                    FEHLER
                  </span>

                  <!-- Warnings tooltip -->
                  <div v-if="row.warnings.length > 0 && !row.isDuplicate" class="group relative">
                    <AlertTriangle class="h-4 w-4 text-amber-500 dark:text-amber-400" />
                    <div class="hidden group-hover:block absolute left-0 top-6 z-10 bg-popover border rounded-lg shadow-lg p-3 w-64">
                      <ul class="text-xs text-foreground space-y-1">
                        <li v-for="(warning, idx) in row.warnings" :key="idx">
                          - {{ warning }}
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>

                <!-- Status info text -->
                <div v-if="row.isDuplicate && !mergeRows.has(row.index)" class="text-xs text-amber-600 dark:text-amber-300 mt-1">
                  Kind existiert bereits
                </div>
                <div v-if="row.isDuplicate && mergeRows.has(row.index) && !rowHasConflicts(row)" class="text-xs text-blue-600 dark:text-blue-300 mt-1">
                  Eltern werden hinzugefügt
                </div>
                <div v-if="row.isDuplicate && mergeRows.has(row.index) && rowHasConflicts(row)" class="text-xs text-blue-600 dark:text-blue-300 mt-1">
                  {{ row.fieldConflicts?.length }} Feld{{ row.fieldConflicts?.length !== 1 ? 'er' : '' }} können aktualisiert werden
                </div>
              </td>

              <!-- Member number -->
              <td class="px-4 py-3 font-mono">
                <template v-if="editingRow === row.index">
                  <input
                    v-model="editedData!.memberNumber"
                    type="text"
                    class="w-20 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                  />
                </template>
                <template v-else>
                  {{ row.child.memberNumber || '-' }}
                </template>
              </td>

              <!-- Name -->
              <td class="px-4 py-3 font-medium">
                <template v-if="editingRow === row.index">
                  <div class="flex gap-1">
                    <input
                      v-model="editedData!.firstName"
                      type="text"
                      placeholder="Vorname"
                      class="w-20 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                    />
                    <input
                      v-model="editedData!.lastName"
                      type="text"
                      placeholder="Nachname"
                      class="w-24 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                    />
                  </div>
                </template>
                <template v-else>
                  <div class="flex items-center gap-2">
                    <span>{{ row.child.firstName }} {{ row.child.lastName }}</span>
                    <!-- Show info icon for duplicate - details shown in expandable row below -->
                    <span 
                      v-if="row.isDuplicate && row.existingChild"
                      class="inline-flex items-center justify-center w-4 h-4 text-xs bg-blue-100 dark:bg-blue-950/40 text-blue-600 dark:text-blue-300 rounded-full cursor-help"
                      title="Bestehendes Kind - Details siehe unten"
                    >i</span>
                  </div>
                </template>
              </td>

              <!-- Birth date -->
              <td class="px-4 py-3">
                <template v-if="editingRow === row.index">
                  <input
                    v-model="editedData!.birthDate"
                    type="text"
                    placeholder="DD.MM.YYYY"
                    class="w-24 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                  />
                </template>
                <template v-else>
                  {{ row.child.birthDate || '-' }}
                </template>
              </td>

              <!-- Entry date -->
              <td class="px-4 py-3">
                <template v-if="editingRow === row.index">
                  <input
                    v-model="editedData!.entryDate"
                    type="text"
                    placeholder="DD.MM.YYYY"
                    class="w-24 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                  />
                </template>
                <template v-else>
                  {{ row.child.entryDate || '-' }}
                </template>
              </td>

              <!-- Legal hours (Rechtsanspruch) - always weekly -->
              <td class="px-4 py-3">
                <template v-if="editingRow === row.index">
                  <input
                    v-model.number="editedData!.legalHours"
                    type="number"
                    placeholder="Std/Woche"
                    class="w-16 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                  />
                </template>
                <template v-else>
                  <span v-if="row.child.legalHours" class="text-foreground">
                    {{ row.child.legalHours }} Std
                  </span>
                  <span v-else class="text-muted-foreground">-</span>
                </template>
              </td>

              <!-- Care hours (Betreuungszeit) - may need conversion from daily to weekly -->
              <td class="px-4 py-3">
                <template v-if="editingRow === row.index">
                  <input
                    v-model.number="editedData!.careHours"
                    type="number"
                    placeholder="Std/Woche"
                    class="w-16 px-2 py-1 text-xs border rounded focus:ring-1 focus:ring-primary"
                  />
                </template>
                <template v-else>
                  <div v-if="row.child.careHours" class="text-foreground">
                    <span v-if="row.child.careHours < 12" class="text-amber-600 dark:text-amber-300" :title="`Umgerechnet von ${row.child.careHours} Std/Tag`">
                      {{ row.child.careHours * 5 }} Std
                      <span class="text-xs text-muted-foreground">({{ row.child.careHours }}/Tag)</span>
                    </span>
                    <span v-else>
                      {{ row.child.careHours }} Std
                    </span>
                  </div>
                  <span v-else class="text-muted-foreground">-</span>
                </template>
              </td>

              <!-- Parent 1 -->
              <td class="px-4 py-3">
                <div v-if="row.parent1 && row.parent1.firstName" class="space-y-1">
                  <div class="font-medium flex items-center gap-2">
                    {{ row.parent1.firstName }} {{ row.parent1.lastName }}
                    <!-- Already linked badge -->
                    <span 
                      v-if="row.parent1.alreadyLinked"
                      class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded bg-muted text-muted-foreground"
                      title="Bereits mit diesem Kind verknüpft"
                    >
                      <Link class="h-3 w-3" />
                      Verknüpft
                    </span>
                  </div>
                  <!-- Only show select if not already linked -->
                  <div v-if="!row.parent1.alreadyLinked && row.parent1.existingMatches && row.parent1.existingMatches.length > 0" class="flex items-center gap-2">
                    <select
                      :value="getParentDecision(row.index, 1)?.action === 'link' ? getParentDecision(row.index, 1)?.existingParentId : 'create'"
                      @change="($event.target as HTMLSelectElement).value === 'create' 
                        ? setParentDecision(row.index, 1, 'create') 
                        : setParentDecision(row.index, 1, 'link', ($event.target as HTMLSelectElement).value)"
                      class="text-xs px-2 py-1 border rounded"
                    >
                      <option value="create">
                        <Plus class="h-3 w-3 inline" />
                        Neu anlegen
                      </option>
                      <option
                        v-for="match in row.parent1.existingMatches"
                        :key="match.id"
                        :value="match.id"
                      >
                        Verknüpfen: {{ match.firstName }} {{ match.lastName }}{{ match.email ? ` (${match.email})` : '' }}
                      </option>
                    </select>
                  </div>
                </div>
                <span v-else class="text-muted-foreground">-</span>
              </td>

              <!-- Parent 2 -->
              <td class="px-4 py-3">
                <div v-if="row.parent2 && row.parent2.firstName" class="space-y-1">
                  <div class="font-medium flex items-center gap-2">
                    {{ row.parent2.firstName }} {{ row.parent2.lastName }}
                    <!-- Already linked badge -->
                    <span 
                      v-if="row.parent2.alreadyLinked"
                      class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded bg-muted text-muted-foreground"
                      title="Bereits mit diesem Kind verknüpft"
                    >
                      <Link class="h-3 w-3" />
                      Verknüpft
                    </span>
                  </div>
                  <!-- Only show select if not already linked -->
                  <div v-if="!row.parent2.alreadyLinked && row.parent2.existingMatches && row.parent2.existingMatches.length > 0" class="flex items-center gap-2">
                    <select
                      :value="getParentDecision(row.index, 2)?.action === 'link' ? getParentDecision(row.index, 2)?.existingParentId : 'create'"
                      @change="($event.target as HTMLSelectElement).value === 'create' 
                        ? setParentDecision(row.index, 2, 'create') 
                        : setParentDecision(row.index, 2, 'link', ($event.target as HTMLSelectElement).value)"
                      class="text-xs px-2 py-1 border rounded"
                    >
                      <option value="create">Neu anlegen</option>
                      <option
                        v-for="match in row.parent2.existingMatches"
                        :key="match.id"
                        :value="match.id"
                      >
                        Verknüpfen: {{ match.firstName }} {{ match.lastName }}{{ match.email ? ` (${match.email})` : '' }}
                      </option>
                    </select>
                  </div>
                </div>
                <span v-else class="text-muted-foreground">-</span>
              </td>

              <!-- Actions -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-1">
                  <template v-if="editingRow === row.index">
                    <button
                      @click="saveEditing(row)"
                      class="p-1 text-green-600 dark:text-green-300 hover:bg-green-50 dark:hover:bg-green-900/40 rounded"
                      title="Speichern"
                    >
                      <Check class="h-4 w-4" />
                    </button>
                    <button
                      @click="cancelEditing"
                      class="p-1 text-muted-foreground hover:bg-accent rounded"
                      title="Abbrechen"
                    >
                      <X class="h-4 w-4" />
                    </button>
                  </template>
                  <template v-else>
                    <button
                      @click="startEditing(row)"
                      class="p-1 text-muted-foreground hover:text-primary hover:bg-accent rounded"
                      title="Bearbeiten"
                    >
                      <Pencil class="h-4 w-4" />
                    </button>
                  </template>
                </div>
              </td>
            </tr>
            <!-- Existing child info expansion row (shown for duplicates not yet in merge mode) -->
            <tr 
              v-if="row.isDuplicate && row.existingChild && !mergeRows.has(row.index)"
              :key="`${row.index}-existing`"
              class="bg-amber-50 dark:bg-amber-950/40 border-t border-amber-100"
            >
              <td colspan="11" class="px-8 py-3">
                <div class="text-sm">
                  <div class="font-medium text-amber-800 dark:text-amber-300 mb-2 flex items-center gap-2">
                    <AlertCircle class="h-4 w-4" />
                    Bestehendes Kind in Datenbank:
                  </div>
                  <div class="bg-card rounded-lg px-4 py-3 border border-amber-200">
                    <dl class="grid grid-cols-2 md:grid-cols-4 gap-x-6 gap-y-2 text-sm">
                      <div>
                        <dt class="text-muted-foreground">Name</dt>
                        <dd class="font-medium">{{ row.existingChild.firstName }} {{ row.existingChild.lastName }}</dd>
                      </div>
                      <div>
                        <dt class="text-muted-foreground">Geburtsdatum</dt>
                        <dd>{{ row.existingChild.birthDate }}</dd>
                      </div>
                      <div>
                        <dt class="text-muted-foreground">Eintrittsdatum</dt>
                        <dd>{{ row.existingChild.entryDate }}</dd>
                      </div>
                      <div v-if="row.existingChild.legalHours || row.existingChild.careHours">
                        <dt class="text-muted-foreground">Betreuung</dt>
                        <dd>
                          <span v-if="row.existingChild.legalHours">{{ row.existingChild.legalHours }} Std RA</span>
                          <span v-if="row.existingChild.legalHours && row.existingChild.careHours"> / </span>
                          <span v-if="row.existingChild.careHours">{{ row.existingChild.careHours }} Std</span>
                        </dd>
                      </div>
                    </dl>
                    <p class="mt-3 text-xs text-amber-700 dark:text-amber-300 border-t border-amber-200 pt-2">
                      Klicke auf "Merge?" um Eltern aus der CSV zu diesem Kind hinzuzufügen.
                    </p>
                  </div>
                </div>
              </td>
            </tr>
            <!-- Field conflicts expansion row -->
            <tr 
              v-if="row.isDuplicate && mergeRows.has(row.index) && rowHasConflicts(row)"
              :key="`${row.index}-conflicts`"
              class="bg-blue-50 dark:bg-blue-950/40 border-t border-blue-100"
            >
              <td colspan="11" class="px-8 py-3">
                <div class="text-sm">
                  <div class="font-medium text-blue-800 dark:text-blue-300 mb-2 flex items-center gap-2">
                    <AlertTriangle class="h-4 w-4" />
                    Unterschiede zwischen CSV und Datenbank:
                  </div>
                  <div class="grid gap-2">
                    <div 
                      v-for="conflict in row.fieldConflicts" 
                      :key="conflict.field"
                      class="flex items-center gap-4 bg-card rounded-lg px-3 py-2 border"
                    >
                      <span class="text-muted-foreground w-32">{{ conflict.fieldLabel }}:</span>
                      <label class="flex items-center gap-2 cursor-pointer">
                        <input
                          type="radio"
                          :name="`conflict-${row.index}-${conflict.field}`"
                          :checked="getConflictResolution(row.index, conflict.field) === 'existing'"
                          @change="setConflictResolution(row.index, conflict.field, 'existing')"
                          class="text-blue-600 dark:text-blue-300"
                        />
                        <span class="text-foreground">
                          <span class="font-medium">Behalten:</span> {{ conflict.existingValue || '-' }}
                        </span>
                      </label>
                      <label class="flex items-center gap-2 cursor-pointer">
                        <input
                          type="radio"
                          :name="`conflict-${row.index}-${conflict.field}`"
                          :checked="getConflictResolution(row.index, conflict.field) === 'new'"
                          @change="setConflictResolution(row.index, conflict.field, 'new')"
                          class="text-blue-600 dark:text-blue-300"
                        />
                        <span class="text-blue-700 dark:text-blue-300">
                          <span class="font-medium">CSV verwenden:</span> {{ conflict.newValue }}
                        </span>
                      </label>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Navigation -->
    <div class="flex justify-between">
      <button
        @click="goBack"
        class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors flex items-center gap-2"
      >
        <ArrowLeft class="h-4 w-4" />
        Zurück
      </button>
      <button
        @click="executeImport"
        :disabled="selectedValidCount === 0 || isLoading"
        class="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
        <template v-else>
          <Upload class="h-4 w-4" />
          <span v-if="mergeRowsCount > 0 && selectedValidCount - mergeRowsCount > 0">
            {{ selectedValidCount - mergeRowsCount }} importieren, {{ mergeRowsCount }} zusammenführen
          </span>
          <span v-else-if="mergeRowsCount > 0">
            {{ mergeRowsCount }} zusammenführen
          </span>
          <span v-else>
            {{ selectedValidCount }} Kinder importieren
          </span>
        </template>
      </button>
    </div>
  </div>
</template>
