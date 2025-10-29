<template>
  <div class="restore-form">
    <h2>Restore Backup</h2>
    
    <div v-if="error" class="alert alert-error">
      {{ error }}
    </div>

    <div v-if="success" class="alert alert-success">
      Restore started successfully! Operation ID: {{ operationId }}
    </div>

    <form @submit.prevent="handleSubmit">
      <div class="form-group">
        <label for="backupFile">Backup File *</label>
        <select
          id="backupFile"
          v-model="selectedBackup"
          class="form-input"
          required
        >
          <option value="">-- Select a backup file --</option>
          <option
            v-for="backup in backups"
            :key="backup.name"
            :value="backup.name"
          >
            {{ backup.name }} ({{ formatSize(backup.size) }}, {{ formatDate(backup.modTime) }})
          </option>
        </select>
      </div>

      <DataTypeSelector v-model="dataTypes" />

      <div class="form-actions">
        <button
          type="submit"
          class="btn btn-primary"
          :disabled="isSubmitting || !selectedBackup || !hasSelectedTypes"
        >
          <span v-if="isSubmitting">Restoring...</span>
          <span v-else>Restore Backup</span>
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue';
import DataTypeSelector from './DataTypeSelector.vue';
import {type BackupFile, listBackups, startRestore} from '../services/api';
import type {DataTypeSelection, RestoreRequest} from '../types/api';

const emit = defineEmits<{
  'restore-started': [operationId: string];
}>();

const backups = ref<BackupFile[]>([]);
const selectedBackup = ref('');
const dataTypes = ref<DataTypeSelection>({
  chats: true,
  prompts: true,
  tools: false,
  files: true,
  models: true,
  knowledge: true,
  users: false,
  groups: false,
  feedbacks: false,
});

const isSubmitting = ref(false);
const error = ref<string | null>(null);
const success = ref(false);
const operationId = ref<string | null>(null);

const hasSelectedTypes = computed(() => {
  return Object.values(dataTypes.value).some((selected) => selected);
});

const formatSize = (bytes: number | undefined): string => {
  if (bytes === undefined || bytes === null) {
    return '0 B';
  }
  
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let unitIndex = 0;
  
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  
  return `${size.toFixed(2)} ${units[unitIndex]}`;
};

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleString();
};

const loadBackups = async () => {
  try {
    backups.value = await listBackups();
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load backups';
  }
};

const handleSubmit = async () => {
  if (!selectedBackup.value) {
    error.value = 'Please select a backup file';
    return;
  }

  if (!hasSelectedTypes.value) {
    error.value = 'Please select at least one data type to restore';
    return;
  }

  isSubmitting.value = true;
  error.value = null;
  success.value = false;

  try {
    const request: RestoreRequest = {
      inputFilename: selectedBackup.value,
      decryptIdentity: '',
      dataTypes: dataTypes.value,
      overwrite: true,
    };

    const response = await startRestore(request);
    operationId.value = response.operationId;
    success.value = true;
    
    // Emit event to parent
    emit('restore-started', response.operationId);

    // Reset form after short delay
    setTimeout(() => {
      success.value = false;
      selectedBackup.value = '';
    }, 3000);
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to start restore';
  } finally {
    isSubmitting.value = false;
  }
};

onMounted(() => {
  loadBackups();
});
</script>

<style scoped>
.restore-form {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  border: 1px solid #e9ecef;
  height: 100%;
}

.restore-form h2 {
  margin: 0 0 1rem 0;
  color: #212529;
  font-size: 1.25rem;
  font-weight: 600;
}

.alert {
  padding: 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
}

.alert-error {
  background: #fee;
  color: #c33;
  border: 1px solid #fcc;
}

.alert-success {
  background: #efe;
  color: #3c3;
  border: 1px solid #cfc;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.375rem;
  font-weight: 600;
  color: #495057;
  font-size: 0.8125rem;
}

.form-input {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.9375rem;
  transition: all 0.2s;
}

.form-input:hover {
  border-color: #adb5bd;
}

.form-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

select.form-input {
  cursor: pointer;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.btn {
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 6px;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
