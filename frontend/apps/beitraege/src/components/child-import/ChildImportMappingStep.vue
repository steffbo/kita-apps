<script setup lang="ts">
import { ArrowLeft, ArrowRight, Loader2, AlertCircle, FileSpreadsheet, Users } from 'lucide-vue-next';
import { useChildImportWizardContext } from '@/composables/useChildImportWizard';

const {
  isLoading,
  parseResult,
  mapping,
  childFields,
  parent1Fields,
  parent2Fields,
  allRequiredFieldsMapped,
  allNewChildFieldsMapped,
  setMapping,
  getSampleValue,
  goToPreview,
  goBack,
} = useChildImportWizardContext();
</script>

<template>
  <div class="space-y-6">
    <!-- Detected info -->
    <div class="card p-6">
      <div class="flex items-center gap-2 mb-4">
        <FileSpreadsheet class="h-5 w-5 text-primary" />
        <h2 class="text-base font-semibold">Datei erkannt</h2>
      </div>
      <div class="grid grid-cols-3 gap-4 text-sm">
        <div>
          <span class="text-muted-foreground">Gefundene Spalten:</span>
          <span class="ml-2 font-medium">{{ parseResult?.headers.length || 0 }}</span>
        </div>
        <div>
          <span class="text-muted-foreground">Datenzeilen:</span>
          <span class="ml-2 font-medium">{{ parseResult?.totalRows || 0 }}</span>
        </div>
        <div>
          <span class="text-muted-foreground">Trennzeichen:</span>
          <span class="ml-2 font-medium font-mono">
            {{ parseResult?.detectedSeparator === ';' ? 'Semikolon (;)' :
               parseResult?.detectedSeparator === ',' ? 'Komma (,)' :
               parseResult?.detectedSeparator === '\t' ? 'Tab' : parseResult?.detectedSeparator }}
          </span>
        </div>
      </div>
    </div>

    <!-- Mapping sections -->
    <div class="card p-6">
      <h2 class="text-base font-semibold mb-4">Feldzuordnung</h2>
      <p class="text-sm text-muted-foreground mb-6">
        Ordne die CSV-Spalten den Systemfeldern zu. Felder mit * sind Pflichtfelder.
      </p>

      <!-- Child fields -->
      <div class="mb-8">
        <h3 class="text-sm font-medium text-foreground mb-3 flex items-center gap-2">
          <Users class="h-4 w-4" />
          Kind
        </h3>
        <div class="grid grid-cols-2 gap-4">
          <div v-for="field in childFields" :key="field.key" class="flex items-center gap-3">
            <label :for="`mapping-${field.key}`" class="label w-40">
              {{ field.label }}
              <span v-if="field.required" class="text-destructive">*</span>
            </label>
            <select
              :id="`mapping-${field.key}`"
              :value="mapping[field.key]"
              @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
              class="input min-w-0"
            >
              <option value="">-- Nicht zuordnen --</option>
              <option
                v-for="(header, index) in parseResult?.headers"
                :key="index"
                :value="index"
              >
                {{ header }} ({{ getSampleValue(index) || '-' }})
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- Parent 1 fields -->
      <div class="mb-8">
        <h3 class="text-sm font-medium text-foreground mb-3 flex items-center gap-2">
          <Users class="h-4 w-4" />
          Elternteil 1 (optional)
        </h3>
        <div class="grid grid-cols-2 gap-4">
          <div v-for="field in parent1Fields" :key="field.key" class="flex items-center gap-3">
            <label :for="`mapping-${field.key}`" class="label w-40">
              {{ field.label.replace('Elternteil 1 - ', '') }}
            </label>
            <select
              :id="`mapping-${field.key}`"
              :value="mapping[field.key]"
              @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
              class="input min-w-0"
            >
              <option value="">-- Nicht zuordnen --</option>
              <option
                v-for="(header, index) in parseResult?.headers"
                :key="index"
                :value="index"
              >
                {{ header }} ({{ getSampleValue(index) || '-' }})
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- Parent 2 fields -->
      <div>
        <h3 class="text-sm font-medium text-foreground mb-3 flex items-center gap-2">
          <Users class="h-4 w-4" />
          Elternteil 2 (optional)
        </h3>
        <div class="grid grid-cols-2 gap-4">
          <div v-for="field in parent2Fields" :key="field.key" class="flex items-center gap-3">
            <label :for="`mapping-${field.key}`" class="label w-40">
              {{ field.label.replace('Elternteil 2 - ', '') }}
            </label>
            <select
              :id="`mapping-${field.key}`"
              :value="mapping[field.key]"
              @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
              class="input min-w-0"
            >
              <option value="">-- Nicht zuordnen --</option>
              <option
                v-for="(header, index) in parseResult?.headers"
                :key="index"
                :value="index"
              >
                {{ header }} ({{ getSampleValue(index) || '-' }})
              </option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- Info box when not all fields for new children are mapped -->
    <div v-if="allRequiredFieldsMapped && !allNewChildFieldsMapped" class="note note-info">
      <AlertCircle />
      <div>
        <h4 class="font-medium">Nur Aktualisierung möglich</h4>
        <p class="text-sm mt-1">
          Nicht alle Pflichtfelder für neue Kinder sind zugeordnet (Vorname, Nachname, Geburtsdatum, Eintrittsdatum).
          Der Import kann nur bestehende Kinder anhand der Mitgliedsnummer aktualisieren.
          Neue Kinder können nicht angelegt werden.
        </p>
      </div>
    </div>

    <!-- Navigation -->
    <div class="flex justify-between">
      <button
        @click="goBack"
        class="btn btn-secondary"
      >
        <ArrowLeft class="h-4 w-4" />
        Zurück
      </button>
      <button
        @click="goToPreview"
        :disabled="!allRequiredFieldsMapped || isLoading"
        class="btn btn-primary"
      >
        <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
        <template v-else>
          Vorschau
          <ArrowRight class="h-4 w-4" />
        </template>
      </button>
    </div>
  </div>
</template>
