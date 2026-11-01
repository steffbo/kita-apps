<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, Eye, FileText, Trash2, Upload } from 'lucide-vue-next';
import { formatDate, formatFileSize } from '@/utils/format';

// Signed contract of a plan (PDF, stored by the backend). Every action reloads through `changed`.
const props = defineProps<{ detail: InstallmentDetail }>();
const emit = defineEmits<{ changed: [] }>();

const busy = ref<'upload' | 'view' | 'remove' | null>(null);
const error = ref<string | null>(null);
const confirmRemove = ref(false);

async function run(action: 'upload' | 'remove', task: () => Promise<void>): Promise<void> {
  busy.value = action;
  error.value = null;
  try {
    await task();
    emit('changed');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Vertrag konnte nicht bearbeitet werden';
  } finally {
    busy.value = null;
  }
}

function upload(event: Event): void {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  void run('upload', () => api.uploadInstallmentContract(props.detail.plan.id, file));
}

function remove(): void {
  confirmRemove.value = false;
  void run('remove', () => api.deleteInstallmentContract(props.detail.plan.id));
}

// The tab opens synchronously with the click so the popup blocker allows it.
async function view(): Promise<void> {
  const tab = window.open('', '_blank');
  busy.value = 'view';
  error.value = null;
  try {
    const file = await api.getInstallmentContract(props.detail.plan.id);
    const url = URL.createObjectURL(file.blob);
    if (tab) tab.location.href = url;
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (e) {
    tab?.close();
    error.value = e instanceof Error ? e.message : 'PDF konnte nicht geladen werden';
  } finally {
    busy.value = null;
  }
}
</script>

<template>
  <div class="space-y-3">
    <div v-if="detail.contract" class="card">
      <div class="flex items-center gap-3 px-4 py-3">
        <span class="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-primary-soft text-primary-text">
          <FileText class="h-4 w-4" />
        </span>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{{ detail.contract.filename }}</p>
          <p class="text-xs text-muted-foreground">
            PDF · {{ formatFileSize(detail.contract.sizeBytes) }} · hochgeladen {{ formatDate(detail.contract.uploadedAt) }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy !== null" @click="view">
          <Eye />Ansehen
        </button>
      </div>
      <div class="flex flex-wrap justify-end gap-2 border-t px-4 py-2.5">
        <label class="btn btn-secondary btn-sm cursor-pointer" :class="busy && 'pointer-events-none opacity-50'">
          <Upload />Ersetzen
          <input type="file" accept="application/pdf" class="hidden" @change="upload" />
        </label>
        <template v-if="confirmRemove">
          <span class="self-center text-xs text-muted-foreground">Vertrag entfernen?</span>
          <button type="button" class="btn btn-danger btn-sm" :disabled="busy !== null" @click="remove">Ja</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="confirmRemove = false">Nein</button>
        </template>
        <button v-else type="button" class="btn btn-danger-outline btn-sm" :disabled="busy !== null"
          @click="confirmRemove = true"><Trash2 />Entfernen</button>
      </div>
    </div>

    <template v-else>
      <div class="note"><AlertTriangle /><span>Noch kein unterschriebener Vertrag hinterlegt.</span></div>
      <div>
        <label class="btn btn-secondary cursor-pointer" :class="busy && 'pointer-events-none opacity-50'">
          <Upload />Vertrag hochladen
          <input type="file" accept="application/pdf" class="hidden" @change="upload" />
        </label>
        <p class="mt-1.5 text-xs text-muted-foreground">PDF, max. 10 MB</p>
      </div>
    </template>

    <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
  </div>
</template>
