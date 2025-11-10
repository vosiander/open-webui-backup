<template>
  <div class="restore-form">
    <h2>Restore Backup</h2>

    <form @submit.prevent="handleSubmit">
      <div class="form-group">
        <label>Upload Backup File</label>
        <div class="upload-container">
          <input
            type="file"
            ref="fileInput"
            accept=".age,.zip"
            @change="handleFileSelected"
            class="file-input"
            :disabled="isUploading"
          />
          <button
            type="button"
            @click="triggerFileInput"
            class="btn btn-secondary"
            :disabled="isUploading"
          >
            <span v-if="isUploading">Uploading...</span>
            <span v-else>Choose File to Upload</span>
          </button>
          <span v-if="selectedFile" class="file-name">{{ selectedFile.name }}</span>
        </div>
      </div>

      <div class="divider">
        <span>OR</span>
      </div>

      <div class="form-group">
        <label for="backupFile">Select Existing Backup *</label>
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
          type="button"
          @click="handleVerify"
          class="btn btn-verify"
          :disabled="isVerifying || !selectedBackup"
        >
          <span v-if="isVerifying">Verifying...</span>
          <span v-else>Verify Backup</span>
        </button>
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
import {type BackupFile, listBackups, startRestore, uploadBackup, verifyBackup} from '../services/api';
import type {DataTypeSelection, RestoreRequest} from '../types/api';

const props = defineProps<{
  ageIdentity?: string;
}>();

const emit = defineEmits<{
  'operation-started': [payload: { operationId: string; type: string }];
  'operation-error': [payload: { message: string; type: string }];
  'operation-success': [payload: { message: string; type: string }];
}>();

const backups = ref<BackupFile[]>([]);
const selectedBackup = ref('');
const fileInput = ref<HTMLInputElement | null>(null);
const selectedFile = ref<File | null>(null);
const isUploading = ref(false);
const isVerifying = ref(false);
const verificationStatus = ref<{ success: boolean; message: string } | null>(null);
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

const triggerFileInput = () => {
  fileInput.value?.click();
};

const handleFileSelected = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];

  if (!file) return;

  selectedFile.value = file;

  // Validate file extension
  if (!file.name.endsWith('.age') && !file.name.endsWith('.zip')) {
    emit('operation-error', {
      message: 'Only .age and .zip files are allowed',
      type: 'restore'
    });
    selectedFile.value = null;
    return;
  }

  // Upload immediately
  isUploading.value = true;

  try {
    await uploadBackup(file);

    // Refresh backup list
    await loadBackups();

    // Clear file input
    selectedFile.value = null;
    if (fileInput.value) {
      fileInput.value.value = '';
    }
  } catch (err) {
    emit('operation-error', {
      message: err instanceof Error ? err.message : 'Failed to upload file',
      type: 'restore'
    });
  } finally {
    isUploading.value = false;
  }
};

const handleVerify = async () => {
  if (!selectedBackup.value) {
    emit('operation-error', {
      message: 'Please select a backup file to verify',
      type: 'restore'
    });
    return;
  }

  isVerifying.value = true;
  verificationStatus.value = null;

  try {
    const result = await verifyBackup(selectedBackup.value, props.ageIdentity || '');
    verificationStatus.value = result;

    if (!result.success) {
      emit('operation-error', {
        message: result.message,
        type: 'restore'
      });
    } else {
      emit('operation-success', {
        message: result.message,
        type: 'restore'
      });
    }
  } catch (err) {
    const errorMsg = err instanceof Error ? err.message : 'Failed to verify backup';
    verificationStatus.value = {
      success: false,
      message: errorMsg
    };
    emit('operation-error', {
      message: errorMsg,
      type: 'restore'
    });
  } finally {
    isVerifying.value = false;
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

.upload-container {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.file-input {
  display: none;
}

.btn-secondary {
  background: #6c757d;
  color: white;
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 6px;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary:hover:not(:disabled) {
  background: #5a6268;
}

.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.file-name {
  color: #495057;
  font-size: 0.875rem;
}

.divider {
  display: flex;
  align-items: center;
  text-align: center;
  margin: 1.5rem 0;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  border-bottom: 1px solid #dee2e6;
}

.divider span {
  padding: 0 1rem;
  color: #6c757d;
  font-size: 0.875rem;
  font-weight: 600;
}

.btn-verify {
  background: #28a745;
  color: white;
}

.btn-verify:hover:not(:disabled) {
  background: #218838;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}
</style>
