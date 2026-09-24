// State and actions of the child CSV import wizard (upload → mapping → preview → result).
// The page provides it; step components inject it via useChildImportWizardContext().
import { ref, computed, onMounted, onUnmounted, inject, type InjectionKey, type Ref } from 'vue';
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

export function useChildImportWizard() {
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
  const previewResult: Ref<ChildImportPreviewResult | null> = ref(null);
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
    legalHours?: number | null;
    careHours?: number | null;
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

  return {
    router,
    currentStep,
    isLoading,
    error,
    isDragging,
    parseResult,
    fileContent,
    mapping,
    previewResult,
    selectedRows,
    parentDecisions,
    existingMemberNumbers,
    mergeRows,
    conflictResolutions,
    executeResult,
    editingRow,
    editedData,
    systemFields,
    childFields,
    parent1Fields,
    parent2Fields,
    allRequiredFieldsMapped,
    allNewChildFieldsMapped,
    selectedValidCount,
    mergeRowsCount,
    sortedPreviewRows,
    handleKeydown,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleFileSelect,
    uploadFile,
    readFileAsBase64,
    autoDetectMapping,
    setMapping,
    getSampleValue,
    goToPreview,
    toggleRow,
    toggleMerge,
    selectAll,
    deselectAll,
    getParentDecisionKey,
    setParentDecision,
    getParentDecision,
    getConflictKey,
    setConflictResolution,
    getConflictResolution,
    rowHasConflicts,
    startEditing,
    cancelEditing,
    saveEditing,
    executeImport,
    goBack,
    finishImport,
  };
}

export type ChildImportWizard = ReturnType<typeof useChildImportWizard>;

export const childImportWizardKey: InjectionKey<ChildImportWizard> = Symbol('childImportWizard');

export function useChildImportWizardContext(): ChildImportWizard {
  const wizard = inject(childImportWizardKey);
  if (!wizard) throw new Error('useChildImportWizardContext() needs a provided child import wizard');
  return wizard;
}
