<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type {
  ChildImportParseResult,
  ChildImportPreviewResult,
  ChildImportPreviewRow,
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
  AlertCircle,
  FileSpreadsheet,
  Users,
  CheckCircle,
  XCircle,
  Plus,
  Pencil,
  X,
  GitMerge,
  Link,
  RefreshCw,
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
// Track member numbers that exist in the database (detected as duplicates on initial preview)
const existingMemberNumbers = ref<Set<string>>(new Set());
// Track which duplicate rows should merge parents into existing child
const mergeRows = ref<Set<number>>(new Set());
// Track field conflict resolutions: Map<"rowIndex-field", "existing" | "new">
const conflictResolutions = ref<Map<string, 'existing' | 'new'>>(new Map());

// Step 4: Results
const executeResult = ref<ChildImportExecuteResult | null>(null);

// Inline editing state
const editingRow = ref<number | null>(null);
const editedData = ref<{
  memberNumber: string;
  firstName: string;
  lastName: string;
  birthDate: string;
  entryDate: string;
  legalHours?: number;
  careHours?: number;
} | null>(null);

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
// For preview, we only REQUIRE member number - other fields are needed only for NEW children
// The backend will validate each row and show which ones need more data
const allRequiredFieldsMapped = computed(() => {
  // Member number is always required to identify the child
  return mapping.value['memberNumber'] !== undefined;
});

// Check if all fields for creating NEW children are mapped
const allNewChildFieldsMapped = computed(() => {
  const requiredForNew = ['memberNumber', 'firstName', 'lastName', 'birthDate', 'entryDate'];
  return requiredForNew.every(key => mapping.value[key] !== undefined);
});

// Get the count of selected valid rows (including merge rows)
const selectedValidCount = computed(() => {
  if (!previewResult.value) return 0;
  return previewResult.value.rows.filter(r => 
    (selectedRows.value.has(r.index) && r.isValid && !r.isDuplicate) ||
    mergeRows.value.has(r.index)
  ).length;
});

// Get the count of merge rows
const mergeRowsCount = computed(() => mergeRows.value.size);

// Sort preview rows: problems (invalid/duplicate without merge) first, then mergeable, then valid rows
const sortedPreviewRows = computed(() => {
  if (!previewResult.value) return [];
  return [...previewResult.value.rows].sort((a, b) => {
    const aIsMerge = mergeRows.value.has(a.index);
    const bIsMerge = mergeRows.value.has(b.index);
    const aHasProblems = (!a.isValid || a.isDuplicate) && !aIsMerge;
    const bHasProblems = (!b.isValid || b.isDuplicate) && !bIsMerge;
    
    // Problems first (duplicates not marked for merge)
    if (aHasProblems && !bHasProblems) return -1;
    if (!aHasProblems && bHasProblems) return 1;
    
    // Within same category, sort by original index
    return a.index - b.index;
  });
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
  
  // Mapping rules with patterns - order matters for tie-breaking but the actual matching
  // is done by finding exact or best matches for each header
  const mappingRules: [string, string[]][] = [
    // Parent 1 fields
    ['parent1FirstName', ['elternteil 1 vorname', 'eltern 1 vorname', 'mutter vorname', 'vater vorname', 'erz1 vorname', 'eltern1vorname']],
    ['parent1LastName', ['elternteil 1 nachname', 'eltern 1 nachname', 'mutter nachname', 'vater nachname', 'erz1 nachname', 'eltern1nachname']],
    ['parent1Email', ['elternteil 1 email', 'eltern 1 email', 'mutter email', 'vater email', 'erz1 email', 'email 1']],
    ['parent1Phone', ['elternteil 1 telefon', 'eltern 1 telefon', 'mutter telefon', 'vater telefon', 'erz1 telefon', 'telefon 1']],
    // Parent 2 fields
    ['parent2FirstName', ['elternteil 2 vorname', 'eltern 2 vorname', 'erz2 vorname', 'eltern2vorname']],
    ['parent2LastName', ['elternteil 2 nachname', 'eltern 2 nachname', 'erz2 nachname', 'eltern2nachname']],
    ['parent2Email', ['elternteil 2 email', 'eltern 2 email', 'erz2 email', 'email 2']],
    ['parent2Phone', ['elternteil 2 telefon', 'eltern 2 telefon', 'erz2 telefon', 'telefon 2']],
    // Child fields
    ['memberNumber', ['mitgliedsnummer', 'mitglieds-nr', 'mitgliedsnr', 'member', 'nr']],
    ['firstName', ['vorname', 'first', 'firstname', 'kind vorname']],
    ['lastName', ['nachname', 'name', 'last', 'lastname', 'kind nachname', 'familienname']],
    ['birthDate', ['geburtsdatum', 'geburtstag', 'birth', 'geb']],
    ['entryDate', ['eintrittsdatum', 'eintritt', 'entry', 'aufnahme', 'start']],
    ['street', ['straße', 'strasse', 'street']],
    ['streetNo', ['hausnummer', 'hausnr', 'haus-nr', 'nr.']],
    ['postalCode', ['plz', 'postleitzahl', 'postal']],
    ['city', ['ort', 'stadt', 'city', 'wohnort']],
    ['legalHours', ['rechtsanspruch', 'legal']],
    ['careHours', ['betreuungszeit', 'betreuung', 'stunden']],
  ];

  // Track which headers have been mapped to avoid duplicate mapping
  const mappedHeaders = new Set<number>();

  headers.forEach((header, index) => {
    const normalizedHeader = header.toLowerCase().trim();
    let bestMatch: { field: string; matchLength: number } | null = null;
    
    for (const [field, keywords] of mappingRules) {
      if (newMapping[field] !== undefined) continue;
      
      for (const keyword of keywords) {
        // Only match if header contains keyword (not the other way around!)
        // This ensures "Vorname" doesn't match "elternteil 1 vorname" just because
        // the keyword contains "vorname"
        if (normalizedHeader.includes(keyword)) {
          // Prefer longer keyword matches (more specific)
          if (!bestMatch || keyword.length > bestMatch.matchLength) {
            bestMatch = { field, matchLength: keyword.length };
          }
        }
      }
    }
    
    if (bestMatch && !mappedHeaders.has(index)) {
      newMapping[bestMatch.field] = index;
      mappedHeaders.add(index);
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

    // Track member numbers that were detected as duplicates (exist in database)
    existingMemberNumbers.value = new Set(
      result.rows
        .filter(r => r.isDuplicate)
        .map(r => r.child.memberNumber)
    );

    // Pre-select all valid rows
    selectedRows.value = new Set(
      result.rows.filter(r => r.isValid && !r.isDuplicate).map(r => r.index)
    );

    // Initialize parent decisions
    parentDecisions.value = new Map();
    
    // Reset merge rows
    mergeRows.value = new Set();
    
    // Reset conflict resolutions
    conflictResolutions.value = new Map();

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

function toggleMerge(index: number) {
  if (mergeRows.value.has(index)) {
    mergeRows.value.delete(index);
  } else {
    mergeRows.value.add(index);
  }
  mergeRows.value = new Set(mergeRows.value);
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

// Conflict resolution helpers
function getConflictKey(rowIndex: number, field: string): string {
  return `${rowIndex}-${field}`;
}

function setConflictResolution(rowIndex: number, field: string, resolution: 'existing' | 'new') {
  const key = getConflictKey(rowIndex, field);
  conflictResolutions.value.set(key, resolution);
  conflictResolutions.value = new Map(conflictResolutions.value);
}

function getConflictResolution(rowIndex: number, field: string): 'existing' | 'new' {
  return conflictResolutions.value.get(getConflictKey(rowIndex, field)) || 'existing';
}

// Check if row has any conflicts to resolve
function rowHasConflicts(row: ChildImportPreviewRow): boolean {
  return row.fieldConflicts !== undefined && row.fieldConflicts.length > 0;
}

// Inline editing handlers
function startEditing(row: ChildImportPreviewRow) {
  editingRow.value = row.index;
  editedData.value = {
    memberNumber: row.child.memberNumber,
    firstName: row.child.firstName,
    lastName: row.child.lastName,
    birthDate: row.child.birthDate,
    entryDate: row.child.entryDate,
    legalHours: row.child.legalHours,
    careHours: row.child.careHours,
  };
}

function cancelEditing() {
  editingRow.value = null;
  editedData.value = null;
}

function saveEditing(row: ChildImportPreviewRow) {
  if (!editedData.value || !previewResult.value) return;
  
  // Find and update the row in previewResult
  const rowIndex = previewResult.value.rows.findIndex(r => r.index === row.index);
  if (rowIndex !== -1) {
    const currentRow = previewResult.value.rows[rowIndex];
    const oldMemberNumber = currentRow.child.memberNumber;
    const newMemberNumber = editedData.value.memberNumber;
    const memberNumberChanged = oldMemberNumber !== newMemberNumber;
    
    currentRow.child = {
      ...currentRow.child,
      memberNumber: newMemberNumber,
      firstName: editedData.value.firstName,
      lastName: editedData.value.lastName,
      birthDate: editedData.value.birthDate,
      entryDate: editedData.value.entryDate,
      legalHours: editedData.value.legalHours,
      careHours: editedData.value.careHours,
    };
    
    // If member number was changed, re-check duplicate status
    if (memberNumberChanged) {
      // Remove old duplicate-related warnings
      currentRow.warnings = currentRow.warnings.filter(w => 
        !w.toLowerCase().includes('existiert bereits') && 
        !w.toLowerCase().includes('duplikat') &&
        !w.toLowerCase().includes('mitgliedsnummer')
      );
      
      // Check if new member number exists in database
      const existsInDatabase = existingMemberNumbers.value.has(newMemberNumber);
      
      // Check if new member number exists in other rows of this import
      const existsInOtherRows = previewResult.value.rows.some(r => 
        r.index !== row.index && r.child.memberNumber === newMemberNumber
      );
      
      if (existsInDatabase) {
        currentRow.isDuplicate = true;
        currentRow.warnings.push(`Kind mit Mitgliedsnummer ${newMemberNumber} existiert bereits`);
        // Deselect the row since it's now a duplicate
        selectedRows.value.delete(row.index);
        selectedRows.value = new Set(selectedRows.value);
      } else if (existsInOtherRows) {
        currentRow.isDuplicate = true;
        currentRow.warnings.push(`Mitgliedsnummer ${newMemberNumber} wird bereits in einer anderen Zeile verwendet`);
        // Deselect the row since it's now a duplicate
        selectedRows.value.delete(row.index);
        selectedRows.value = new Set(selectedRows.value);
      } else {
        // No longer a duplicate
        currentRow.isDuplicate = false;
        currentRow.existingChildId = undefined;
      }
    }
    
    // Re-validate the row (basic validation)
    const child = currentRow.child;
    const isValid = !!(child.memberNumber && child.firstName && child.lastName && child.birthDate && child.entryDate);
    currentRow.isValid = isValid;
    
    // Auto-select the row if it's now valid and not a duplicate
    if (isValid && !currentRow.isDuplicate && !selectedRows.value.has(row.index)) {
      selectedRows.value.add(row.index);
      selectedRows.value = new Set(selectedRows.value);
    }
    
    // Update counts
    previewResult.value.validCount = previewResult.value.rows.filter(r => r.isValid && !r.isDuplicate).length;
    previewResult.value.errorCount = previewResult.value.rows.filter(r => !r.isValid || r.isDuplicate).length;
  }
  
  editingRow.value = null;
  editedData.value = null;
}

async function executeImport() {
  if (!previewResult.value) return;

  isLoading.value = true;
  error.value = null;

  try {
    // Build import request - include both new rows and merge rows
    const selectedPreviewRows = previewResult.value.rows.filter(r => 
      selectedRows.value.has(r.index) && r.isValid && !r.isDuplicate
    );
    
    const mergePreviewRows = previewResult.value.rows.filter(r =>
      mergeRows.value.has(r.index) && r.isDuplicate && r.existingChildId
    );

    const rows = [
      // New children
      ...selectedPreviewRows.map(r => ({
        index: r.index,
        child: {
          ...r.child,
          // Normalize care hours: if < 12, it's daily hours, multiply by 5 for weekly
          careHours: r.child.careHours && r.child.careHours < 12 
            ? r.child.careHours * 5 
            : r.child.careHours,
        },
        parent1: r.parent1,
        parent2: r.parent2,
      })),
      // Merge rows - add parents to existing children
      ...mergePreviewRows.map(r => {
        // Build field updates from conflict resolutions
        const fieldUpdates: Record<string, string> = {};
        if (r.fieldConflicts) {
          for (const conflict of r.fieldConflicts) {
            const resolution = getConflictResolution(r.index, conflict.field);
            if (resolution === 'new') {
              fieldUpdates[conflict.field] = conflict.newValue;
            }
          }
        }
        
        return {
          index: r.index,
          child: {
            ...r.child,
            careHours: r.child.careHours && r.child.careHours < 12 
              ? r.child.careHours * 5 
              : r.child.careHours,
          },
          parent1: r.parent1,
          parent2: r.parent2,
          existingChildId: r.existingChildId,
          mergeParents: true,
          fieldUpdates: Object.keys(fieldUpdates).length > 0 ? fieldUpdates : undefined,
        };
      }),
    ];
    
    const allRowsToProcess = [...selectedPreviewRows, ...mergePreviewRows];

    // Collect parent decisions
    const decisions: ParentDecision[] = [];
    for (const row of allRowsToProcess) {
      if (row.parent1 && row.parent1.firstName && row.parent1.lastName && !row.parent1.alreadyLinked) {
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
      if (row.parent2 && row.parent2.firstName && row.parent2.lastName && !row.parent2.alreadyLinked) {
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

      <!-- Info box when not all fields for new children are mapped -->
      <div v-if="allRequiredFieldsMapped && !allNewChildFieldsMapped" class="p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <div class="flex items-start gap-3">
          <AlertCircle class="h-5 w-5 text-blue-500 flex-shrink-0 mt-0.5" />
          <div>
            <h4 class="font-medium text-blue-900">Nur Aktualisierung möglich</h4>
            <p class="text-sm text-blue-700 mt-1">
              Nicht alle Pflichtfelder für neue Kinder sind zugeordnet (Vorname, Nachname, Geburtsdatum, Eintrittsdatum).
              Der Import kann nur bestehende Kinder anhand der Mitgliedsnummer aktualisieren.
              Neue Kinder können nicht angelegt werden.
            </p>
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
              {{ previewResult?.errorCount || 0 }} mit Fehlern/Duplikaten
            </p>
          </div>
          <div class="flex items-center gap-4">
            <button @click="selectAll" class="text-sm text-primary hover:underline">
              Alle auswählen
            </button>
            <button @click="deselectAll" class="text-sm text-gray-600 hover:underline">
              Alle abwählen
            </button>
            <div class="text-sm font-medium text-gray-900">
              {{ selectedValidCount - mergeRowsCount }} neu,
              <span v-if="mergeRowsCount > 0" class="text-blue-600">{{ mergeRowsCount }} Merge</span>
              <span v-else class="text-gray-500">0 Merge</span>
            </div>
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
                <th class="px-4 py-3 text-left font-medium text-gray-500">Rechtsanspr.</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Betreuung</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Elternteil 1</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">Elternteil 2</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500 w-20"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              <template
                v-for="row in sortedPreviewRows"
                :key="row.index"
              >
              <tr
                :class="[
                  'hover:bg-gray-50',
                  // Invalid rows: red
                  !row.isValid && !row.isDuplicate ? 'bg-red-50' : '',
                  // Duplicates not marked for merge: amber/yellow
                  row.isDuplicate && !mergeRows.has(row.index) ? 'bg-amber-50' : '',
                  // Duplicates marked for merge: blue
                  row.isDuplicate && mergeRows.has(row.index) ? 'bg-blue-50' : '',
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
                    class="h-4 w-4 text-primary rounded border-gray-300 focus:ring-primary"
                  />
                  <!-- For duplicates with existing child: merge toggle -->
                  <button
                    v-else-if="row.isDuplicate && row.existingChildId"
                    @click="toggleMerge(row.index)"
                    :class="[
                      'flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors',
                      mergeRows.has(row.index)
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-200 text-gray-700 hover:bg-gray-300',
                    ]"
                    :title="mergeRows.has(row.index) ? 'Zusammenführung deaktivieren' : 'Eltern zu bestehendem Kind hinzufügen'"
                  >
                    <GitMerge class="h-3 w-3" />
                    {{ mergeRows.has(row.index) ? 'Merge' : 'Merge?' }}
                  </button>
                  <!-- For invalid rows: disabled indicator -->
                  <span v-else class="text-gray-400 text-xs">-</span>
                </td>

                <!-- Status -->
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <!-- Action badge -->
                    <span 
                      v-if="row.action === 'create' && row.isValid && !row.isDuplicate"
                      class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-green-100 text-green-700"
                    >
                      <Plus class="h-3 w-3" />
                      NEU
                    </span>
                    <span 
                      v-else-if="row.isDuplicate && mergeRows.has(row.index) && rowHasConflicts(row)"
                      class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-blue-100 text-blue-700"
                    >
                      <RefreshCw class="h-3 w-3" />
                      UPDATE
                    </span>
                    <span 
                      v-else-if="row.isDuplicate && mergeRows.has(row.index)"
                      class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-blue-100 text-blue-700"
                    >
                      <GitMerge class="h-3 w-3" />
                      MERGE
                    </span>
                    <span 
                      v-else-if="row.isDuplicate"
                      class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-amber-100 text-amber-700"
                    >
                      EXISTIERT
                    </span>
                    <span 
                      v-else-if="!row.isValid"
                      class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-red-100 text-red-700"
                    >
                      <XCircle class="h-3 w-3" />
                      FEHLER
                    </span>
                    
                    <!-- Warnings tooltip -->
                    <div v-if="row.warnings.length > 0 && !row.isDuplicate" class="group relative">
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
                  
                  <!-- Status info text -->
                  <div v-if="row.isDuplicate && !mergeRows.has(row.index)" class="text-xs text-amber-600 mt-1">
                    Kind existiert bereits
                  </div>
                  <div v-if="row.isDuplicate && mergeRows.has(row.index) && !rowHasConflicts(row)" class="text-xs text-blue-600 mt-1">
                    Eltern werden hinzugefügt
                  </div>
                  <div v-if="row.isDuplicate && mergeRows.has(row.index) && rowHasConflicts(row)" class="text-xs text-blue-600 mt-1">
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
                        class="inline-flex items-center justify-center w-4 h-4 text-xs bg-blue-100 text-blue-600 rounded-full cursor-help"
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
                    <span v-if="row.child.legalHours" class="text-gray-700">
                      {{ row.child.legalHours }} Std
                    </span>
                    <span v-else class="text-gray-400">-</span>
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
                    <div v-if="row.child.careHours" class="text-gray-700">
                      <span v-if="row.child.careHours < 12" class="text-amber-600" :title="`Umgerechnet von ${row.child.careHours} Std/Tag`">
                        {{ row.child.careHours * 5 }} Std
                        <span class="text-xs text-gray-500">({{ row.child.careHours }}/Tag)</span>
                      </span>
                      <span v-else>
                        {{ row.child.careHours }} Std
                      </span>
                    </div>
                    <span v-else class="text-gray-400">-</span>
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
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded bg-gray-100 text-gray-600"
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
                  <span v-else class="text-gray-400">-</span>
                </td>

                <!-- Parent 2 -->
                <td class="px-4 py-3">
                  <div v-if="row.parent2 && row.parent2.firstName" class="space-y-1">
                    <div class="font-medium flex items-center gap-2">
                      {{ row.parent2.firstName }} {{ row.parent2.lastName }}
                      <!-- Already linked badge -->
                      <span 
                        v-if="row.parent2.alreadyLinked"
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded bg-gray-100 text-gray-600"
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
                  <span v-else class="text-gray-400">-</span>
                </td>

                <!-- Actions -->
                <td class="px-4 py-3">
                  <div class="flex items-center gap-1">
                    <template v-if="editingRow === row.index">
                      <button
                        @click="saveEditing(row)"
                        class="p-1 text-green-600 hover:bg-green-50 rounded"
                        title="Speichern"
                      >
                        <Check class="h-4 w-4" />
                      </button>
                      <button
                        @click="cancelEditing"
                        class="p-1 text-gray-600 hover:bg-gray-100 rounded"
                        title="Abbrechen"
                      >
                        <X class="h-4 w-4" />
                      </button>
                    </template>
                    <template v-else>
                      <button
                        @click="startEditing(row)"
                        class="p-1 text-gray-400 hover:text-primary hover:bg-gray-100 rounded"
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
                class="bg-amber-50/50 border-t border-amber-100"
              >
                <td colspan="11" class="px-8 py-3">
                  <div class="text-sm">
                    <div class="font-medium text-amber-800 mb-2 flex items-center gap-2">
                      <AlertCircle class="h-4 w-4" />
                      Bestehendes Kind in Datenbank:
                    </div>
                    <div class="bg-white rounded-lg px-4 py-3 border border-amber-200">
                      <dl class="grid grid-cols-2 md:grid-cols-4 gap-x-6 gap-y-2 text-sm">
                        <div>
                          <dt class="text-gray-500">Name</dt>
                          <dd class="font-medium">{{ row.existingChild.firstName }} {{ row.existingChild.lastName }}</dd>
                        </div>
                        <div>
                          <dt class="text-gray-500">Geburtsdatum</dt>
                          <dd>{{ row.existingChild.birthDate }}</dd>
                        </div>
                        <div>
                          <dt class="text-gray-500">Eintrittsdatum</dt>
                          <dd>{{ row.existingChild.entryDate }}</dd>
                        </div>
                        <div v-if="row.existingChild.legalHours || row.existingChild.careHours">
                          <dt class="text-gray-500">Betreuung</dt>
                          <dd>
                            <span v-if="row.existingChild.legalHours">{{ row.existingChild.legalHours }} Std RA</span>
                            <span v-if="row.existingChild.legalHours && row.existingChild.careHours"> / </span>
                            <span v-if="row.existingChild.careHours">{{ row.existingChild.careHours }} Std</span>
                          </dd>
                        </div>
                      </dl>
                      <p class="mt-3 text-xs text-amber-700 border-t border-amber-200 pt-2">
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
                class="bg-blue-50/50 border-t border-blue-100"
              >
                <td colspan="11" class="px-8 py-3">
                  <div class="text-sm">
                    <div class="font-medium text-blue-800 mb-2 flex items-center gap-2">
                      <AlertTriangle class="h-4 w-4" />
                      Unterschiede zwischen CSV und Datenbank:
                    </div>
                    <div class="grid gap-2">
                      <div 
                        v-for="conflict in row.fieldConflicts" 
                        :key="conflict.field"
                        class="flex items-center gap-4 bg-white rounded-lg px-3 py-2 border"
                      >
                        <span class="text-gray-600 w-32">{{ conflict.fieldLabel }}:</span>
                        <label class="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            :name="`conflict-${row.index}-${conflict.field}`"
                            :checked="getConflictResolution(row.index, conflict.field) === 'existing'"
                            @change="setConflictResolution(row.index, conflict.field, 'existing')"
                            class="text-blue-600"
                          />
                          <span class="text-gray-700">
                            <span class="font-medium">Behalten:</span> {{ conflict.existingValue || '-' }}
                          </span>
                        </label>
                        <label class="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            :name="`conflict-${row.index}-${conflict.field}`"
                            :checked="getConflictResolution(row.index, conflict.field) === 'new'"
                            @change="setConflictResolution(row.index, conflict.field, 'new')"
                            class="text-blue-600"
                          />
                          <span class="text-blue-700">
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
