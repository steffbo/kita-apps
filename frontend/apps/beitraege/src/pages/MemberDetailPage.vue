<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Member, UpdateMemberRequest } from '@/api/types';
import {
  AlertTriangle,
  Edit,
  Trash2,
  Loader2,
  Calendar,
  MapPin,
  Check,
  Mail,
  Phone,
  Hash,
} from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import { formatDate, formatDateForInput } from '@/utils/format';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const member = ref<Member | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateMemberRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

const memberId = computed(() => route.params.id as string);

async function loadMember() {
  isLoading.value = true;
  error.value = null;
  try {
    member.value = await api.getMember(memberId.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadMember);

function openEditDialog() {
  if (!member.value) return;
  editForm.value = {
    firstName: member.value.firstName,
    lastName: member.value.lastName,
    email: member.value.email,
    phone: member.value.phone,
    street: member.value.street,
    streetNo: member.value.streetNo,
    postalCode: member.value.postalCode,
    city: member.value.city,
    membershipStart: formatDateForInput(member.value.membershipStart),
    membershipEnd: member.value.membershipEnd ? formatDateForInput(member.value.membershipEnd) : undefined,
    isActive: member.value.isActive,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!member.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateMember(memberId.value, editForm.value);
    member.value = updated;
    showEditDialog.value = false;
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
}

async function handleDelete() {
  isDeleting.value = true;
  try {
    await api.deleteMember(memberId.value);
    router.push('/mitglieder');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

async function handleToggleActive() {
  if (!member.value) return;
  isEditing.value = true;
  try {
    const updated = await api.updateMember(memberId.value, { isActive: !member.value.isActive });
    member.value = updated;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Aktualisieren';
  } finally {
    isEditing.value = false;
  }
}
</script>

<template>
  <div>
    <PageHeader :title="member ? `${member.firstName} ${member.lastName}` : 'Mitglied'" back="/mitglieder">
      <template #meta v-if="member">
        <span :class="member.isActive ? 'badge badge-success' : 'badge'">
          {{ member.isActive ? 'Aktiv' : 'Inaktiv' }}
        </span>
      </template>
      <template #actions v-if="member">
        <button type="button" class="btn btn-secondary" :disabled="isEditing" @click="handleToggleActive">
          {{ member?.isActive ? 'Deaktivieren' : 'Aktivieren' }}
        </button>
        <button type="button" class="btn btn-secondary btn-icon" title="Bearbeiten" aria-label="Bearbeiten" @click="openEditDialog"><Edit /></button>
        <button v-if="authStore.isAdmin" type="button" class="btn btn-danger-outline btn-icon" title="Löschen" aria-label="Löschen" @click="showDeleteDialog = true"><Trash2 /></button>
      </template>
    </PageHeader>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <AlertTriangle />
      <div>
        <p class="text-destructive">{{ error }}</p>
      <button @click="loadMember" class="btn btn-secondary btn-sm">
        Erneut versuchen
      </button>
      </div>
    </div>

    <!-- Member details -->
    <div v-else-if="member">
      <!-- Header -->
      <div class="card p-6 mb-6">

        <!-- Info grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          <div class="flex items-start gap-3">
            <Hash class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Mitgliedsnummer</p>
              <p class="font-medium font-mono">{{ member.memberNumber }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Mitglied seit</p>
              <p class="font-medium">{{ formatDate(member.membershipStart) }}</p>
            </div>
          </div>
          <div v-if="member.membershipEnd" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Mitgliedschaft endet</p>
              <p class="font-medium">{{ formatDate(member.membershipEnd) }}</p>
            </div>
          </div>
          <div v-if="member.email" class="flex items-start gap-3">
            <Mail class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">E-Mail</p>
              <a :href="`mailto:${member.email}`" class="font-medium text-primary hover:underline">
                {{ member.email }}
              </a>
            </div>
          </div>
          <div v-if="member.phone" class="flex items-start gap-3">
            <Phone class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Telefon</p>
              <a :href="`tel:${member.phone}`" class="font-medium text-primary hover:underline">
                {{ member.phone }}
              </a>
            </div>
          </div>
          <div v-if="member.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Adresse</p>
              <p class="font-medium">{{ member.street }} {{ member.streetNo }}</p>
              <p class="text-sm text-muted-foreground">{{ member.postalCode }} {{ member.city }}</p>
            </div>
          </div>
        </div>

        <!-- Timestamps -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex gap-6 text-sm text-muted-foreground">
            <span>Erstellt: {{ formatDate(member.createdAt) }}</span>
            <span>Aktualisiert: {{ formatDate(member.updatedAt) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Dialog -->
    <AppSheet v-if="showEditDialog" title="Mitglied bearbeiten" @close="showEditDialog = false">

        <form id="edit-member-form" @submit.prevent="handleEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-firstName" class="label">Vorname</label>
              <input
                id="edit-firstName"
                v-model="editForm.firstName"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="edit-lastName" class="label">Nachname</label>
              <input
                id="edit-lastName"
                v-model="editForm.lastName"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div>
            <label for="edit-email" class="label">E-Mail</label>
            <input
              id="edit-email"
              v-model="editForm.email"
              type="email"
              class="input"
            />
          </div>

          <div>
            <label for="edit-phone" class="label">Telefon</label>
            <input
              id="edit-phone"
              v-model="editForm.phone"
              type="tel"
              class="input"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="edit-street" class="label">Straße</label>
              <input
                id="edit-street"
                v-model="editForm.street"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="edit-streetNo" class="label">Hausnr.</label>
              <input
                id="edit-streetNo"
                v-model="editForm.streetNo"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-postalCode" class="label">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="input"
              />
            </div>
            <div class="col-span-2">
              <label for="edit-city" class="label">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="input"
              />
            </div>
          </div>

          <!-- Membership dates -->
          <div class="pt-4 border-t">
            <h3 class="text-sm font-medium text-foreground mb-3">Mitgliedschaft</h3>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="edit-membershipStart" class="label">Mitglied seit</label>
                <input
                  id="edit-membershipStart"
                  v-model="editForm.membershipStart"
                  type="date"
                  class="input"
                />
              </div>
              <div>
                <label for="edit-membershipEnd" class="label">Endet am</label>
                <input
                  id="edit-membershipEnd"
                  v-model="editForm.membershipEnd"
                  type="date"
                  class="input"
                />
              </div>
            </div>
          </div>

          <div v-if="editError" class="note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ editError }}</p>
          </div>


        </form>

    <template #footer><div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showEditDialog = false"
              class="btn btn-secondary"
            >
              Abbrechen
            </button>
            <button form="edit-member-form" type="submit"
              :disabled="isEditing"
              class="btn btn-primary"
            >
              <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div></template></AppSheet>

    <!-- Delete Confirmation Dialog -->
    <AppSheet v-if="showDeleteDialog" title="Mitglied löschen?" @close="showDeleteDialog = false">

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ member?.firstName }} {{ member?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden.
        </p>



    <template #footer><div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            @click="handleDelete"
            :disabled="isDeleting"
            class="btn btn-danger"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div></template></AppSheet>
  </div>
</template>
