<template>
  <div class="backup-form">
    <h2>Create Backup</h2>

    <form @submit.prevent="handleSubmit">
      <DataTypeSelector v-model="dataTypes" />

      <div class="form-group">
        <label for="outputFilename">Backup Filename *</label>
        <input
          id="outputFilename"
          v-model="outputFilename"
          type="text"
          class="form-input"
          placeholder="e.g., backup-2025-10-29-21-25.age"
          required
        />
        <small class="form-hint">.age extension will be added automatically if not present</small>
      </div>

      <div class="form-group">
        <label for="description">Description (Optional)</label>
        <input
          id="description"
          v-model="description"
          type="text"
          class="form-input"
          placeholder="e.g., Weekly backup before update"
        />
      </div>

      <div class="form-actions">
        <button
          type="submit"
          class="btn btn-primary"
          :disabled="isSubmitting || !hasSelectedTypes"
        >
          <span v-if="isSubmitting">Creating Backup...</span>
          <span v-else>Create Backup</span>
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue';
import DataTypeSelector from './DataTypeSelector.vue';
import {startBackup} from '../services/api';
import type {BackupRequest, DataTypeSelection} from '../types/api';

const emit = defineEmits<{
  'operation-started': [payload: { operationId: string; type: string }];
  'operation-error': [payload: { message: string; type: string }];
}>();

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

// Generate default filename with timestamp
const generateDefaultFilename = (): string => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  const hours = String(now.getHours()).padStart(2, '0');
  const minutes = String(now.getMinutes()).padStart(2, '0');
  return `backup-${year}-${month}-${day}-${hours}-${minutes}.age`;
};

const outputFilename = ref(generateDefaultFilename());
const description = ref('');
const isSubmitting = ref(false);

const hasSelectedTypes = computed(() => {
  return Object.values(dataTypes.value).some((selected) => selected);
});

const handleSubmit = async () => {
  if (!hasSelectedTypes.value) {
    emit('operation-error', {
      message: 'Please select at least one data type to backup',
      type: 'backup'
    });
    return;
  }

  isSubmitting.value = true;

  try {
    const request: BackupRequest = {
      outputFilename: outputFilename.value,
      encryptRecipients: [],
      dataTypes: dataTypes.value,
    };

    const response = await startBackup(request);
    
    // Emit event to parent - App.vue will handle display
    emit('operation-started', {
      operationId: response.operationId,
      type: 'backup'
    });
  } catch (err) {
    emit('operation-error', {
      message: err instanceof Error ? err.message : 'Failed to start backup',
      type: 'backup'
    });
  } finally {
    isSubmitting.value = false;
  }
};
</script>

<style scoped>
.backup-form {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  border: 1px solid #e9ecef;
  height: 100%;
}

.backup-form h2 {
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

.form-hint {
  display: block;
  margin-top: 0.25rem;
  color: #6c757d;
  font-size: 0.75rem;
  font-style: italic;
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
