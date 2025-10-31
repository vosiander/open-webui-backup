<template>
  <div class="restore-form">
    <h2>Restore Backup</h2>

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

const props = defineProps<{
  ageIdentity?: string;
}>();

const emit = defineEmits<{
  'operation-started': [payload: { operationId: string; type: string }];
  'operation-error': [payload: { message: string; type: string }];
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
    emit('operation-error', {
      message: err instanceof Error ? err.message : 'Failed to load backups',
      type: 'restore'
    });
  }
};

const handleSubmit = async () => {
  if (!selectedBackup.value) {
    emit('operation-error', {
      message: 'Please select a backup file',
      type: 'restore'
    });
    return;
  }

  if (!hasSelectedTypes.value) {
    emit('operation-error', {
      message: 'Please select at least one data type to restore',
      type: 'restore'
    });
    return;
  }

  isSubmitting.value = true;

  try {
    const request: RestoreRequest = {
      inputFilename: selectedBackup.value,
      decryptIdentity: props.ageIdentity || '',
      dataTypes: dataTypes.value,
      overwrite: true,
    };

    const response = await startRestore(request);
    
    // Emit event to parent - App.vue will handle display
    emit('operation-started', {
      operationId: response.operationId,
      type: 'restore'
    });

    // Reset form after short delay
    setTimeout(() => {
      selectedBackup.value = '';
    }, 3000);
  } catch (err) {
    emit('operation-error', {
      message: err instanceof Error ? err.message : 'Failed to start restore',
      type: 'restore'
    });
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
