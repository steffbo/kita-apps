<script setup lang="ts">
import { provide } from 'vue';
import { ArrowLeft, Loader2, Check, AlertTriangle, FileSpreadsheet, CheckCircle } from 'lucide-vue-next';
import ChildImportMappingStep from '@/components/child-import/ChildImportMappingStep.vue';
import ChildImportPreviewStep from '@/components/child-import/ChildImportPreviewStep.vue';
import { childImportWizardKey, useChildImportWizard } from '@/composables/useChildImportWizard';

const wizard = useChildImportWizard();
provide(childImportWizardKey, wizard);

const {
  router,
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
    <!-- Header -->
    <div class="mb-6">
      <button
        @click="router.push('/kinder')"
        class="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4"
      >
        <ArrowLeft class="h-4 w-4" />
        Zurück zur Übersicht
      </button>
      
      <h1 class="text-2xl font-bold text-gray-900">Kinder importieren</h1>
      <p class="text-gray-600 mt-1">CSV-Datei hochladen und Kinder anlegen</p>
    </div>

    <!-- Step indicator -->
    <div class="mb-8">
      <div class="flex items-center justify-between">
        <div v-for="step in 4" :key="step" class="flex items-center">
          <div
            :class="[
              'w-10 h-10 rounded-full flex items-center justify-center font-medium transition-colors',
              currentStep >= step
                ? 'bg-primary text-white'
                : 'bg-gray-200 text-gray-500',
            ]"
          >
            <Check v-if="currentStep > step" class="h-5 w-5" />
            <span v-else>{{ step }}</span>
          </div>
          <span
            :class="[
              'ml-2 text-sm font-medium',
              currentStep >= step ? 'text-gray-900' : 'text-gray-500',
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
              currentStep > step ? 'bg-primary' : 'bg-gray-200',
            ]"
          />
        </div>
      </div>
    </div>

    <!-- Error display -->
    <div v-if="error" class="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
      <AlertTriangle class="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-red-700">{{ error }}</p>
        <button @click="error = null" class="text-sm text-red-600 underline mt-1">
          Schließen
        </button>
      </div>
    </div>

    <!-- Step 1: Upload -->
    <div v-if="currentStep === 1" class="bg-white rounded-xl border p-8">
      <div
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
        :class="[
          'border-2 border-dashed rounded-xl p-12 text-center transition-colors',
          isDragging
            ? 'border-primary bg-primary/5'
            : 'border-gray-300 hover:border-gray-400',
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
          <p class="mt-4 text-gray-600">Datei wird verarbeitet...</p>
        </div>
        
        <label v-else for="file-upload" class="cursor-pointer">
          <FileSpreadsheet class="h-12 w-12 text-gray-400 mx-auto" />
          <p class="mt-4 text-lg font-medium text-gray-900">
            CSV-Datei hier ablegen oder klicken zum Auswählen
          </p>
          <p class="mt-2 text-sm text-gray-500">
            Unterstützte Formate: CSV mit Semikolon, Komma oder Tab als Trennzeichen
          </p>
        </label>
      </div>

      <div class="mt-6 p-4 bg-blue-50 rounded-lg">
        <h3 class="font-medium text-blue-900">Hinweise zum CSV-Format</h3>
        <ul class="mt-2 text-sm text-blue-800 space-y-1">
          <li>- Die erste Zeile sollte die Spaltenüberschriften enthalten</li>
          <li>- Pflichtfelder: Mitgliedsnummer, Vorname, Nachname, Geburtsdatum, Eintrittsdatum</li>
          <li>- Datumsformate: DD.MM.YYYY oder YYYY-MM-DD</li>
          <li>- Elterndaten sind optional und können in separaten Spalten angegeben werden</li>
        </ul>
      </div>
    </div>

    <!-- Step 2: Field Mapping -->
    <ChildImportMappingStep v-if="currentStep === 2" />

    <!-- Step 3: Preview -->
    <ChildImportPreviewStep v-if="currentStep === 3" />

    <!-- Step 4: Results -->
    <div v-if="currentStep === 4" class="space-y-6">
      <div class="bg-white rounded-xl border p-8 text-center">
        <div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <CheckCircle class="h-8 w-8 text-green-600" />
        </div>
        <h2 class="text-2xl font-bold text-gray-900 mb-2">Import abgeschlossen</h2>
        <p class="text-gray-600">Die Daten wurden erfolgreich importiert.</p>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-6">
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-primary">{{ executeResult?.childrenCreated || 0 }}</div>
          <div class="text-gray-600 mt-1">Kinder erstellt</div>
        </div>
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-blue-600">{{ executeResult?.childrenUpdated || 0 }}</div>
          <div class="text-gray-600 mt-1">Kinder aktualisiert</div>
        </div>
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-green-600">{{ executeResult?.parentsCreated || 0 }}</div>
          <div class="text-gray-600 mt-1">Eltern erstellt</div>
        </div>
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-amber-600">{{ executeResult?.parentsLinked || 0 }}</div>
          <div class="text-gray-600 mt-1">Eltern verknüpft</div>
        </div>
      </div>

      <!-- Errors -->
      <div v-if="executeResult?.errors && executeResult.errors.length > 0" class="bg-red-50 rounded-xl border border-red-200 p-6">
        <h3 class="font-semibold text-red-800 mb-3 flex items-center gap-2">
          <AlertTriangle class="h-5 w-5" />
          Fehler beim Import ({{ executeResult.errors.length }})
        </h3>
        <ul class="space-y-2 text-sm text-red-700">
          <li v-for="err in executeResult.errors" :key="err.rowIndex">
            Zeile {{ err.rowIndex + 1 }}: {{ err.error }}
          </li>
        </ul>
      </div>

      <!-- Finish -->
      <div class="flex justify-center">
        <button
          @click="finishImport"
          class="px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2"
        >
          <Check class="h-5 w-5" />
          Zur Kinderübersicht
        </button>
      </div>
    </div>
  </div>
</template>
