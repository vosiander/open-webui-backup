<template>
  <div class="config-panel">
    <h3>Configuration</h3>
    
    <div v-if="loading" class="loading">Loading configuration...</div>
    <div v-else-if="loadError" class="error">{{ loadError }}</div>
    
    <div v-else-if="config" class="config-details">
      <div v-if="saveSuccess" class="alert alert-success">
        Configuration saved successfully!
      </div>
      <div v-if="saveError" class="alert alert-error">
        {{ saveError }}
      </div>

      <div class="config-row">
        <div class="config-item">
          <label for="openWebUIURL">Open WebUI URL *</label>
          <input
            id="openWebUIURL"
            v-model="openWebUIURL"
            type="text"
            class="config-input"
            placeholder="https://openwebui.example.com"
            required
          />
        </div>
        
        <div class="config-item">
          <label for="apiKey">API Key</label>
          <input
            id="apiKey"
            v-model="apiKey"
            type="password"
            class="config-input"
            placeholder="Enter API key (optional)"
          />
        </div>
      </div>

      <div class="config-row">
        <div class="config-item">
          <label>Server Port:</label>
          <span class="config-value">{{ config.serverPort }}</span>
        </div>
        
        <div class="config-item">
          <label>Backups Directory:</label>
          <span class="config-value">{{ config.backupsDir }}</span>
        </div>
      </div>

      <div class="config-row config-row-full">
        <div class="config-item">
          <label for="ageIdentity">Age Identity (Private Key):</label>
          <textarea
            id="ageIdentity"
            v-model="ageIdentity"
            class="config-textarea"
            placeholder="Enter age identity key or leave empty to use environment default"
            rows="3"
          ></textarea>
        </div>
      </div>

      <div class="config-row config-row-full">
        <div class="config-item">
          <label for="ageRecipients">Age Recipients (Public Keys):</label>
          <textarea
            id="ageRecipients"
            v-model="ageRecipients"
            class="config-textarea"
            placeholder="Enter age recipient keys (one per line) or leave empty to use environment default"
            rows="3"
          ></textarea>
        </div>
      </div>

      <div class="config-actions">
        <button
          @click="handleSave"
          class="btn btn-primary"
          :disabled="isSaving"
        >
          <span v-if="isSaving">Saving...</span>
          <span v-else>💾 Save Configuration</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {onMounted, ref, watch} from 'vue';
import type {ConfigResponse} from '../types/api';
import {fetchConfig, updateConfig} from '../services/api';

const config = ref<ConfigResponse | null>(null);
const loading = ref(true);
const loadError = ref<string | null>(null);
const openWebUIURL = ref('');
const apiKey = ref('');
const ageIdentity = ref('');
const ageRecipients = ref('');
const isSaving = ref(false);
const saveSuccess = ref(false);
const saveError = ref<string | null>(null);

const emit = defineEmits<{
  'update:ageIdentity': [value: string];
  'update:ageRecipients': [value: string];
}>();

onMounted(async () => {
  try {
    config.value = await fetchConfig();
    
    // Set values from config
    openWebUIURL.value = config.value.openWebUIURL;
    if (config.value.apiKey) {
      apiKey.value = config.value.apiKey;
    }
    
    // Set default values from environment if available
    if (config.value.defaultAgeIdentity) {
      ageIdentity.value = config.value.defaultAgeIdentity;
    }
    if (config.value.defaultAgeRecipients) {
      ageRecipients.value = config.value.defaultAgeRecipients;
    }
    
    // Emit initial values
    emit('update:ageIdentity', ageIdentity.value);
    emit('update:ageRecipients', ageRecipients.value);
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Failed to load configuration';
  } finally {
    loading.value = false;
  }
});

// Watch for changes and emit updates
watch(ageIdentity, (newValue) => {
  emit('update:ageIdentity', newValue);
});

watch(ageRecipients, (newValue) => {
  emit('update:ageRecipients', newValue);
});

const handleSave = async () => {
  if (!openWebUIURL.value) {
    saveError.value = 'Open WebUI URL is required';
    return;
  }

  isSaving.value = true;
  saveSuccess.value = false;
  saveError.value = null;

  try {
    const updatedConfig = await updateConfig({
      openWebUIURL: openWebUIURL.value,
      apiKey: apiKey.value || undefined,
    });
    
    config.value = updatedConfig;
    saveSuccess.value = true;
    
    // Clear success message after 3 seconds
    setTimeout(() => {
      saveSuccess.value = false;
    }, 3000);
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : 'Failed to save configuration';
  } finally {
    isSaving.value = false;
  }
};

// Expose methods to parent
defineExpose({
  getAgeIdentity: () => ageIdentity.value,
  getAgeRecipients: () => ageRecipients.value,
});
</script>

<style scoped>
.config-panel {
  background: #ffffff;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  border: 1px solid #e9ecef;
}

.config-panel h3 {
  margin: 0 0 1rem 0;
  font-size: 1.25rem;
  color: #212529;
  font-weight: 600;
}

.loading {
  padding: 2rem;
  text-align: center;
  color: #6c757d;
}

.error {
  padding: 1rem;
  margin-bottom: 1rem;
  color: #dc3545;
  background: #f8d7da;
  border: 1px solid #f5c6cb;
  border-radius: 4px;
}

.alert {
  padding: 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
  font-weight: 500;
}

.alert-success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.alert-error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.config-details {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.config-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.config-row-full {
  grid-template-columns: 1fr;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.config-item label {
  font-weight: 600;
  color: #495057;
  font-size: 0.8125rem;
  margin-bottom: 0.25rem;
}

.config-input {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.9375rem;
  transition: all 0.2s;
  background: #ffffff;
}

.config-input:hover {
  border-color: #adb5bd;
}

.config-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

.config-value {
  color: #212529;
  padding: 0.625rem 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.8125rem;
  border: 1px solid #e9ecef;
}

.config-textarea {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.8125rem;
  resize: vertical;
  transition: all 0.2s;
  background: #ffffff;
}

.config-textarea:hover {
  border-color: #adb5bd;
}

.config-textarea:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

.config-textarea::placeholder,
.config-input::placeholder {
  color: #6c757d;
  font-style: italic;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.25rem;
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

@media (max-width: 768px) {
  .config-row {
    grid-template-columns: 1fr;
  }
}
</style>
