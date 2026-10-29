<script setup lang="ts">
import { provide } from 'vue';
import { Loader2, Check, AlertTriangle, FileSpreadsheet, CheckCircle } from 'lucide-vue-next';
import PageHeader from '@/components/ui/PageHeader.vue';
import ChildImportMappingStep from '@/components/child-import/ChildImportMappingStep.vue';
import ChildImportPreviewStep from '@/components/child-import/ChildImportPreviewStep.vue';
import { childImportWizardKey, useChildImportWizard } from '@/composables/useChildImportWizard';

const wizard = useChildImportWizard();
provide(childImportWizardKey, wizard);

const {
  currentStep,
  isLoading,
  error,
  isDragging,
  executeResult,
  handleDragOver,
  handleDragLeave,
  handleDrop,
  handleFileSelect,
  finishImport,
} = wizard;
</script>

<template>
  <div class="max-w-6xl mx-auto">
    <PageHeader title="Kinder importieren" back="/kinder" />

    <!-- Step indicator -->
    <div class="mb-8">
      <div class="flex items-center justify-between">
        <div v-for="step in 4" :key="step" class="flex items-center">
          <div
            :class="[
              'w-10 h-10 rounded-lg flex items-center justify-center font-medium transition-colors',
              currentStep >= step
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-muted-foreground',
            ]"
          >
            <Check v-if="currentStep > step" class="h-5 w-5" />
            <span v-else>{{ step }}</span>
          </div>
          <span
            :class="[
              'ml-2 text-sm font-medium',
              currentStep >= step ? 'text-foreground' : 'text-muted-foreground',
            ]"
          >
            {{
              step === 1 ? 'Upload' :
              step === 2 ? 'Zuordnung' :
              step === 3 ? 'Vorschau' : 'Fertig'
            }}
          </span>
          <div
            v-if="step < 4"
            :class="[
              'w-16 h-0.5 mx-4',
              currentStep > step ? 'bg-primary' : 'bg-muted',
            ]"
          />
        </div>
      </div>
    </div>

    <!-- Error display -->
    <div v-if="error" class="note note-danger mb-6">
      <AlertTriangle />
      <div>
        <p>{{ error }}</p>
        <button type="button" @click="error = null" class="btn btn-ghost btn-sm mt-1">Schließen</button>
      </div>
    </div>

    <!-- Step 1: Upload -->
    <div v-if="currentStep === 1" class="card p-8">
      <div
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
        :class="[
          'border-2 border-dashed rounded-xl p-12 text-center transition-colors',
          isDragging
            ? 'border-primary bg-primary/5'
            : 'border-border hover:border-border',
        ]"
      >
        <input
          type="file"
          accept=".csv"
          @change="handleFileSelect"
          class="hidden"
          id="file-upload"
        />
        
        <div v-if="isLoading" class="flex flex-col items-center">
          <Loader2 class="h-12 w-12 text-primary animate-spin" />
          <p class="mt-4 text-muted-foreground">Datei wird verarbeitet...</p>
        </div>
        
        <label v-else for="file-upload" class="cursor-pointer">
          <FileSpreadsheet class="h-12 w-12 text-muted-foreground mx-auto" />
          <p class="mt-4 text-lg font-medium text-foreground">
            CSV-Datei hier ablegen oder klicken zum Auswählen
          </p>
          <p class="mt-2 text-sm text-muted-foreground">
            Unterstützte Formate: CSV mit Semikolon, Komma oder Tab als Trennzeichen
          </p>
        </label>
      </div>

      <div class="note note-info mt-6">
        <FileSpreadsheet />
        <div>
        <h3 class="font-medium">Hinweise zum CSV-Format</h3>
        <ul class="mt-2 space-y-1">
          <li>- Die erste Zeile sollte die Spaltenüberschriften enthalten</li>
          <li>- Pflichtfelder: Mitgliedsnummer, Vorname, Nachname, Geburtsdatum, Eintrittsdatum</li>
          <li>- Datumsformate: DD.MM.YYYY oder YYYY-MM-DD</li>
          <li>- Elterndaten sind optional und können in separaten Spalten angegeben werden</li>
        </ul>
        </div>
      </div>
    </div>

    <!-- Step 2: Field Mapping -->
    <ChildImportMappingStep v-if="currentStep === 2" />

    <!-- Step 3: Preview -->
    <ChildImportPreviewStep v-if="currentStep === 3" />

    <!-- Step 4: Results -->
    <div v-if="currentStep === 4" class="space-y-6">
      <div class="card p-8 text-center">
        <div class="w-16 h-16 rounded-full bg-success-soft flex items-center justify-center mx-auto mb-4">
          <CheckCircle class="h-8 w-8 text-success" />
        </div>
        <h2 class="text-2xl font-semibold text-foreground mb-2">Import abgeschlossen</h2>
        <p class="text-muted-foreground">Die Daten wurden erfolgreich importiert.</p>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-6">
        <div class="card p-6 text-center">
          <div class="text-3xl font-semibold text-primary">{{ executeResult?.childrenCreated || 0 }}</div>
          <div class="text-muted-foreground mt-1">Kinder erstellt</div>
        </div>
        <div class="card p-6 text-center">
          <div class="text-3xl font-semibold text-primary">{{ executeResult?.childrenUpdated || 0 }}</div>
          <div class="text-muted-foreground mt-1">Kinder aktualisiert</div>
        </div>
        <div class="card p-6 text-center">
          <div class="text-3xl font-semibold text-success">{{ executeResult?.parentsCreated || 0 }}</div>
          <div class="text-muted-foreground mt-1">Eltern erstellt</div>
        </div>
        <div class="card p-6 text-center">
          <div class="text-3xl font-semibold text-warning">{{ executeResult?.parentsLinked || 0 }}</div>
          <div class="text-muted-foreground mt-1">Eltern verknüpft</div>
        </div>
      </div>

      <!-- Errors -->
      <div v-if="executeResult?.errors && executeResult.errors.length > 0" class="note note-danger">
        <div class="w-full"><h3 class="font-semibold mb-3 flex items-center gap-2">
          <AlertTriangle class="h-5 w-5" />
          Fehler beim Import ({{ executeResult.errors.length }})
        </h3>
        <ul class="space-y-2 text-sm">
          <li v-for="err in executeResult.errors" :key="err.rowIndex">
            Zeile {{ err.rowIndex + 1 }}: {{ err.error }}
          </li>
        </ul></div>
      </div>

      <!-- Finish -->
      <div class="flex justify-center">
        <button
          @click="finishImport"
          class="btn btn-primary"
        >
          <Check class="h-5 w-5" />
          Zur Kinderübersicht
        </button>
      </div>
    </div>
  </div>
</template>
