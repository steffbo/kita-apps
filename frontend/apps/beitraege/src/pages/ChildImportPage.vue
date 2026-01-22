<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type {
  ChildImportParseResult,
  ChildImportPreviewResult,
  ChildImportExecuteResult,
  ParentDecision,
  SystemField,
} from '@/api/types';
import {
  ArrowLeft,
  ArrowRight,
  Upload,
  Loader2,
  Check,
  AlertTriangle,
  FileSpreadsheet,
  Users,
  CheckCircle,
  XCircle,
  Plus,
} from 'lucide-vue-next';

const router = useRouter();

// Wizard state
const currentStep = ref(1);
const isLoading = ref(false);
const error = ref<string | null>(null);

// Step 1: File upload
const isDragging = ref(false);
const parseResult = ref<ChildImportParseResult | null>(null);
const fileContent = ref<string>(''); // Base64 encoded

// Step 2: Field mapping
const mapping = ref<Record<string, number>>({});

// Step 3: Preview
const previewResult = ref<ChildImportPreviewResult | null>(null);
const selectedRows = ref<Set<number>>(new Set());
const parentDecisions = ref<Map<string, ParentDecision>>(new Map());

// Step 4: Results
const executeResult = ref<ChildImportExecuteResult | null>(null);

// System fields for mapping
const systemFields: SystemField[] = [
  // Child fields
  { key: 'memberNumber', label: 'Mitgliedsnummer', required: true, group: 'child' },
  { key: 'firstName', label: 'Vorname (Kind)', required: true, group: 'child' },
  { key: 'lastName', label: 'Nachname (Kind)', required: true, group: 'child' },
  { key: 'birthDate', label: 'Geburtsdatum', required: true, group: 'child' },
  { key: 'entryDate', label: 'Eintrittsdatum', required: true, group: 'child' },
  { key: 'street', label: 'Straße', required: false, group: 'child' },
  { key: 'streetNo', label: 'Hausnummer', required: false, group: 'child' },
  { key: 'postalCode', label: 'PLZ', required: false, group: 'child' },
  { key: 'city', label: 'Ort', required: false, group: 'child' },
  { key: 'legalHours', label: 'Rechtsanspruch (Std.)', required: false, group: 'child' },
  { key: 'careHours', label: 'Betreuungszeit (Std.)', required: false, group: 'child' },
  // Parent 1 fields
  { key: 'parent1FirstName', label: 'Elternteil 1 - Vorname', required: false, group: 'parent1' },
  { key: 'parent1LastName', label: 'Elternteil 1 - Nachname', required: false, group: 'parent1' },
  { key: 'parent1Email', label: 'Elternteil 1 - E-Mail', required: false, group: 'parent1' },
  { key: 'parent1Phone', label: 'Elternteil 1 - Telefon', required: false, group: 'parent1' },
  // Parent 2 fields
  { key: 'parent2FirstName', label: 'Elternteil 2 - Vorname', required: false, group: 'parent2' },
  { key: 'parent2LastName', label: 'Elternteil 2 - Nachname', required: false, group: 'parent2' },
  { key: 'parent2Email', label: 'Elternteil 2 - E-Mail', required: false, group: 'parent2' },
  { key: 'parent2Phone', label: 'Elternteil 2 - Telefon', required: false, group: 'parent2' },
];

const childFields = computed(() => systemFields.filter(f => f.group === 'child'));
const parent1Fields = computed(() => systemFields.filter(f => f.group === 'parent1'));
const parent2Fields = computed(() => systemFields.filter(f => f.group === 'parent2'));

// Check if all required fields are mapped
const allRequiredFieldsMapped = computed(() => {
  const required = systemFields.filter(f => f.required);
  return required.every(f => mapping.value[f.key] !== undefined);
});

// Get the count of selected valid rows
const selectedValidCount = computed(() => {
  if (!previewResult.value) return 0;
  return previewResult.value.rows.filter(r => 
    selectedRows.value.has(r.index) && r.isValid && !r.isDuplicate
  ).length;
});

// ESC key handler
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    router.push('/kinder');
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

// Step 1: File upload handlers
function handleDragOver(e: DragEvent) {
  e.preventDefault();
  isDragging.value = true;
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault();
  isDragging.value = false;
}

function handleDrop(e: DragEvent) {
  e.preventDefault();
  isDragging.value = false;
  const file = e.dataTransfer?.files[0];
  if (file) {
    uploadFile(file);
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (file) {
    uploadFile(file);
  }
}

async function uploadFile(file: File) {
  if (!file.name.endsWith('.csv')) {
    error.value = 'Bitte nur CSV-Dateien hochladen';
    return;
  }

  isLoading.value = true;
  error.value = null;

  try {
    // Read file content for later use
    const content = await readFileAsBase64(file);
    fileContent.value = content;

    // Parse CSV on server
    const result = await api.parseChildImportCSV(file);
    parseResult.value = result;

    // Auto-detect mapping based on headers
    autoDetectMapping(result.headers);

    // Move to step 2
    currentStep.value = 2;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Upload fehlgeschlagen';
  } finally {
    isLoading.value = false;
  }
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      // Remove data URL prefix if present
      const base64 = result.includes(',') ? result.split(',')[1] : result;
      resolve(base64);
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

function autoDetectMapping(headers: string[]) {
  const newMapping: Record<string, number> = {};
  
  const mappingRules: Record<string, string[]> = {
    memberNumber: ['mitgliedsnummer', 'mitglieds-nr', 'mitgliedsnr', 'member', 'nr'],
    firstName: ['vorname', 'first', 'firstname', 'kind vorname'],
    lastName: ['nachname', 'name', 'last', 'lastname', 'kind nachname', 'familienname'],
    birthDate: ['geburtsdatum', 'geburtstag', 'birth', 'geb'],
    entryDate: ['eintrittsdatum', 'eintritt', 'entry', 'aufnahme', 'start'],
    street: ['straße', 'strasse', 'street'],
    streetNo: ['hausnummer', 'hausnr', 'haus-nr', 'nr.'],
    postalCode: ['plz', 'postleitzahl', 'postal'],
    city: ['ort', 'stadt', 'city', 'wohnort'],
    legalHours: ['rechtsanspruch', 'legal'],
    careHours: ['betreuungszeit', 'betreuung', 'stunden'],
    parent1FirstName: ['elternteil 1 vorname', 'mutter vorname', 'vater vorname', 'erz1 vorname'],
    parent1LastName: ['elternteil 1 nachname', 'mutter nachname', 'vater nachname', 'erz1 nachname'],
    parent1Email: ['elternteil 1 email', 'mutter email', 'vater email', 'erz1 email', 'email 1'],
    parent1Phone: ['elternteil 1 telefon', 'mutter telefon', 'vater telefon', 'erz1 telefon', 'telefon 1'],
    parent2FirstName: ['elternteil 2 vorname', 'erz2 vorname'],
    parent2LastName: ['elternteil 2 nachname', 'erz2 nachname'],
    parent2Email: ['elternteil 2 email', 'erz2 email', 'email 2'],
    parent2Phone: ['elternteil 2 telefon', 'erz2 telefon', 'telefon 2'],
  };

  headers.forEach((header, index) => {
    const normalizedHeader = header.toLowerCase().trim();
    
    for (const [field, keywords] of Object.entries(mappingRules)) {
      if (newMapping[field] !== undefined) continue;
      
      for (const keyword of keywords) {
        if (normalizedHeader.includes(keyword) || keyword.includes(normalizedHeader)) {
          newMapping[field] = index;
          break;
        }
      }
    }
  });

  mapping.value = newMapping;
}

// Step 2: Mapping handlers
function setMapping(field: string, columnIndex: number | undefined) {
  if (columnIndex === undefined) {
    delete mapping.value[field];
  } else {
    mapping.value[field] = columnIndex;
  }
}

function getSampleValue(columnIndex: number): string {
  if (!parseResult.value || !parseResult.value.sampleRows.length) return '';
  return parseResult.value.sampleRows[0][columnIndex] || '';
}

async function goToPreview() {
  if (!allRequiredFieldsMapped.value) {
    error.value = 'Bitte alle Pflichtfelder zuordnen';
    return;
  }

  isLoading.value = true;
  error.value = null;

  try {
    const result = await api.previewChildImport({
      fileContent: fileContent.value,
      separator: parseResult.value?.detectedSeparator || ';',
      mapping: mapping.value,
      skipHeader: true,
    });

    previewResult.value = result;

    // Pre-select all valid rows
    selectedRows.value = new Set(
      result.rows.filter(r => r.isValid && !r.isDuplicate).map(r => r.index)
    );

    // Initialize parent decisions
    parentDecisions.value = new Map();

    currentStep.value = 3;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Vorschau fehlgeschlagen';
  } finally {
    isLoading.value = false;
  }
}

// Step 3: Preview handlers
function toggleRow(index: number) {
  if (selectedRows.value.has(index)) {
    selectedRows.value.delete(index);
  } else {
    selectedRows.value.add(index);
  }
  selectedRows.value = new Set(selectedRows.value);
}

function selectAll() {
  if (!previewResult.value) return;
  selectedRows.value = new Set(
    previewResult.value.rows.filter(r => r.isValid && !r.isDuplicate).map(r => r.index)
  );
}

function deselectAll() {
  selectedRows.value = new Set();
}

function getParentDecisionKey(rowIndex: number, parentIndex: 1 | 2): string {
  return `${rowIndex}-${parentIndex}`;
}

function setParentDecision(rowIndex: number, parentIndex: 1 | 2, action: 'create' | 'link', existingParentId?: string) {
  const key = getParentDecisionKey(rowIndex, parentIndex);
  parentDecisions.value.set(key, {
    rowIndex,
    parentIndex,
    action,
    existingParentId,
  });
  parentDecisions.value = new Map(parentDecisions.value);
}

function getParentDecision(rowIndex: number, parentIndex: 1 | 2): ParentDecision | undefined {
  return parentDecisions.value.get(getParentDecisionKey(rowIndex, parentIndex));
}

async function executeImport() {
  if (!previewResult.value) return;

  isLoading.value = true;
  error.value = null;

  try {
    // Build import request
    const selectedPreviewRows = previewResult.value.rows.filter(r => 
      selectedRows.value.has(r.index) && r.isValid && !r.isDuplicate
    );

    const rows = selectedPreviewRows.map(r => ({
      index: r.index,
      child: r.child,
      parent1: r.parent1,
      parent2: r.parent2,
    }));

    // Collect parent decisions
    const decisions: ParentDecision[] = [];
    for (const row of selectedPreviewRows) {
      if (row.parent1 && row.parent1.firstName && row.parent1.lastName) {
        const decision = getParentDecision(row.index, 1);
        if (decision) {
          decisions.push(decision);
        } else {
          // Default: create new parent
          decisions.push({
            rowIndex: row.index,
            parentIndex: 1,
            action: 'create',
          });
        }
      }
      if (row.parent2 && row.parent2.firstName && row.parent2.lastName) {
        const decision = getParentDecision(row.index, 2);
        if (decision) {
          decisions.push(decision);
        } else {
          decisions.push({
            rowIndex: row.index,
            parentIndex: 2,
            action: 'create',
          });
        }
      }
    }

    const result = await api.executeChildImport({
      rows,
      parentDecisions: decisions,
    });

    executeResult.value = result;
    currentStep.value = 4;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Import fehlgeschlagen';
  } finally {
    isLoading.value = false;
  }
}

function goBack() {
  if (currentStep.value > 1) {
    currentStep.value--;
  } else {
    router.push('/kinder');
  }
}

function finishImport() {
  router.push('/kinder');
}
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
    <div v-if="currentStep === 2" class="space-y-6">
      <!-- Detected info -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center gap-2 mb-4">
          <FileSpreadsheet class="h-5 w-5 text-primary" />
          <h2 class="text-lg font-semibold">Datei erkannt</h2>
        </div>
        <div class="grid grid-cols-3 gap-4 text-sm">
          <div>
            <span class="text-gray-500">Gefundene Spalten:</span>
            <span class="ml-2 font-medium">{{ parseResult?.headers.length || 0 }}</span>
          </div>
          <div>
            <span class="text-gray-500">Datenzeilen:</span>
            <span class="ml-2 font-medium">{{ parseResult?.totalRows || 0 }}</span>
          </div>
          <div>
            <span class="text-gray-500">Trennzeichen:</span>
            <span class="ml-2 font-medium font-mono">
              {{ parseResult?.detectedSeparator === ';' ? 'Semikolon (;)' :
                 parseResult?.detectedSeparator === ',' ? 'Komma (,)' :
                 parseResult?.detectedSeparator === '\t' ? 'Tab' : parseResult?.detectedSeparator }}
            </span>
          </div>
        </div>
      </div>

      <!-- Mapping sections -->
      <div class="bg-white rounded-xl border p-6">
        <h2 class="text-lg font-semibold mb-4">Feldzuordnung</h2>
        <p class="text-sm text-gray-600 mb-6">
          Ordne die CSV-Spalten den Systemfeldern zu. Felder mit * sind Pflichtfelder.
        </p>

        <!-- Child fields -->
        <div class="mb-8">
          <h3 class="text-sm font-medium text-gray-700 mb-3 flex items-center gap-2">
            <Users class="h-4 w-4" />
            Kind
          </h3>
          <div class="grid grid-cols-2 gap-4">
            <div v-for="field in childFields" :key="field.key" class="flex items-center gap-3">
              <label :for="`mapping-${field.key}`" class="w-40 text-sm">
                {{ field.label }}
                <span v-if="field.required" class="text-red-500">*</span>
              </label>
              <select
                :id="`mapping-${field.key}`"
                :value="mapping[field.key]"
                @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
                class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none text-sm"
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
          <h3 class="text-sm font-medium text-gray-700 mb-3 flex items-center gap-2">
            <Users class="h-4 w-4" />
            Elternteil 1 (optional)
          </h3>
          <div class="grid grid-cols-2 gap-4">
            <div v-for="field in parent1Fields" :key="field.key" class="flex items-center gap-3">
              <label :for="`mapping-${field.key}`" class="w-40 text-sm">
                {{ field.label.replace('Elternteil 1 - ', '') }}
              </label>
              <select
                :id="`mapping-${field.key}`"
                :value="mapping[field.key]"
                @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
                class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none text-sm"
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
          <h3 class="text-sm font-medium text-gray-700 mb-3 flex items-center gap-2">
            <Users class="h-4 w-4" />
            Elternteil 2 (optional)
          </h3>
          <div class="grid grid-cols-2 gap-4">
            <div v-for="field in parent2Fields" :key="field.key" class="flex items-center gap-3">
              <label :for="`mapping-${field.key}`" class="w-40 text-sm">
                {{ field.label.replace('Elternteil 2 - ', '') }}
              </label>
              <select
                :id="`mapping-${field.key}`"
                :value="mapping[field.key]"
                @change="setMapping(field.key, ($event.target as HTMLSelectElement).value ? parseInt(($event.target as HTMLSelectElement).value) : undefined)"
                class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none text-sm"
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

      <!-- Navigation -->
      <div class="flex justify-between">
        <button
          @click="goBack"
          class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors flex items-center gap-2"
        >
          <ArrowLeft class="h-4 w-4" />
          Zurück
        </button>
        <button
          @click="goToPreview"
          :disabled="!allRequiredFieldsMapped || isLoading"
          class="px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
          <template v-else>
            Vorschau
            <ArrowRight class="h-4 w-4" />
          </template>
        </button>
      </div>
    </div>

    <!-- Step 3: Preview -->
    <div v-if="currentStep === 3" class="space-y-6">
      <!-- Summary -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold">Vorschau</h2>
            <p class="text-sm text-gray-600 mt-1">
              {{ previewResult?.validCount || 0 }} gültige Einträge,
              {{ previewResult?.errorCount || 0 }} mit Fehlern
            </p>
          </div>
          <div class="flex items-center gap-4">
            <button @click="selectAll" class="text-sm text-primary hover:underline">
              Alle auswählen
            </button>
            <button @click="deselectAll" class="text-sm text-gray-600 hover:underline">
              Alle abwählen
            </button>
            <span class="text-sm font-medium text-gray-900">
              {{ selectedValidCount }} ausgewählt
            </span>
          </div>
        </div>
      </div>

      <!-- Preview table -->
      <div class="bg-white rounded-xl border overflow-hidden">
        <div class="overflow-x-auto max-h-[500px] overflow-y-auto">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 sticky top-0">
              <tr>
                <th class="px-4 py-3 text-left font-medium text-gray-500 w-12"></th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Status</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Mitglieds-Nr.</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Name</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Geburtsdatum</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Eintrittsdatum</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Elternteil 1</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Elternteil 2</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              <tr
                v-for="row in previewResult?.rows"
                :key="row.index"
                :class="[
                  'hover:bg-gray-50',
                  !row.isValid || row.isDuplicate ? 'bg-red-50' : '',
                  selectedRows.has(row.index) && row.isValid && !row.isDuplicate ? 'bg-primary/5' : '',
                ]"
              >
                <!-- Checkbox -->
                <td class="px-4 py-3">
                  <input
                    type="checkbox"
                    :checked="selectedRows.has(row.index)"
                    :disabled="!row.isValid || row.isDuplicate"
                    @change="toggleRow(row.index)"
                    class="h-4 w-4 text-primary rounded border-gray-300 focus:ring-primary disabled:opacity-50"
                  />
                </td>

                <!-- Status -->
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <CheckCircle v-if="row.isValid && !row.isDuplicate" class="h-5 w-5 text-green-500" />
                    <XCircle v-else class="h-5 w-5 text-red-500" />
                    <div v-if="row.warnings.length > 0" class="group relative">
                      <AlertTriangle class="h-4 w-4 text-amber-500" />
                      <div class="hidden group-hover:block absolute left-0 top-6 z-10 bg-white border rounded-lg shadow-lg p-3 w-64">
                        <ul class="text-xs text-gray-700 space-y-1">
                          <li v-for="(warning, idx) in row.warnings" :key="idx">
                            - {{ warning }}
                          </li>
                        </ul>
                      </div>
                    </div>
                  </div>
                </td>

                <!-- Member number -->
                <td class="px-4 py-3 font-mono">{{ row.child.memberNumber || '-' }}</td>

                <!-- Name -->
                <td class="px-4 py-3 font-medium">
                  {{ row.child.firstName }} {{ row.child.lastName }}
                </td>

                <!-- Birth date -->
                <td class="px-4 py-3">{{ row.child.birthDate || '-' }}</td>

                <!-- Entry date -->
                <td class="px-4 py-3">{{ row.child.entryDate || '-' }}</td>

                <!-- Parent 1 -->
                <td class="px-4 py-3">
                  <div v-if="row.parent1 && row.parent1.firstName" class="space-y-1">
                    <div class="font-medium">{{ row.parent1.firstName }} {{ row.parent1.lastName }}</div>
                    <div v-if="row.parent1.existingMatches && row.parent1.existingMatches.length > 0" class="flex items-center gap-2">
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
                  <span v-else class="text-gray-400">-</span>
                </td>

                <!-- Parent 2 -->
                <td class="px-4 py-3">
                  <div v-if="row.parent2 && row.parent2.firstName" class="space-y-1">
                    <div class="font-medium">{{ row.parent2.firstName }} {{ row.parent2.lastName }}</div>
                    <div v-if="row.parent2.existingMatches && row.parent2.existingMatches.length > 0" class="flex items-center gap-2">
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
                  <span v-else class="text-gray-400">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Navigation -->
      <div class="flex justify-between">
        <button
          @click="goBack"
          class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors flex items-center gap-2"
        >
          <ArrowLeft class="h-4 w-4" />
          Zurück
        </button>
        <button
          @click="executeImport"
          :disabled="selectedValidCount === 0 || isLoading"
          class="px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
          <template v-else>
            <Upload class="h-4 w-4" />
            {{ selectedValidCount }} Kinder importieren
          </template>
        </button>
      </div>
    </div>

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
      <div class="grid grid-cols-3 gap-6">
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-primary">{{ executeResult?.childrenCreated || 0 }}</div>
          <div class="text-gray-600 mt-1">Kinder erstellt</div>
        </div>
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-green-600">{{ executeResult?.parentsCreated || 0 }}</div>
          <div class="text-gray-600 mt-1">Eltern erstellt</div>
        </div>
        <div class="bg-white rounded-xl border p-6 text-center">
          <div class="text-3xl font-bold text-blue-600">{{ executeResult?.parentsLinked || 0 }}</div>
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
