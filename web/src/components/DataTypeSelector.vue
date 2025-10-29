<template>
  <div class="data-type-selector">
    <h3>Select Data Types</h3>
    <div class="checkbox-group">
      <label class="checkbox-label">
        <input
          type="checkbox"
          v-model="localSelection.chats"
          @change="emitChange"
        />
        <span class="checkbox-text">
          <strong>Chats</strong>
          <span class="checkbox-description">User chat histories and conversations</span>
        </span>
      </label>

      <label class="checkbox-label">
        <input
          type="checkbox"
          v-model="localSelection.prompts"
          @change="emitChange"
        />
        <span class="checkbox-text">
          <strong>Prompts</strong>
          <span class="checkbox-description">Custom prompts and templates</span>
        </span>
      </label>

      <label class="checkbox-label">
        <input
          type="checkbox"
          v-model="localSelection.files"
          @change="emitChange"
        />
        <span class="checkbox-text">
          <strong>Documents</strong>
          <span class="checkbox-description">Uploaded documents and files</span>
        </span>
      </label>

      <label class="checkbox-label">
        <input
          type="checkbox"
          v-model="localSelection.models"
          @change="emitChange"
        />
        <span class="checkbox-text">
          <strong>Models</strong>
          <span class="checkbox-description">Model configurations</span>
        </span>
      </label>

      <label class="checkbox-label">
        <input
          type="checkbox"
          v-model="localSelection.knowledge"
          @change="emitChange"
        />
        <span class="checkbox-text">
          <strong>Knowledge</strong>
          <span class="checkbox-description">Knowledge base entries</span>
        </span>
      </label>
    </div>

    <div class="selection-actions">
      <button @click="selectAll" class="btn-secondary">Select All</button>
      <button @click="selectNone" class="btn-secondary">Select None</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import {ref, watch} from 'vue';
import type {DataTypeSelection} from '../types/api';

interface Props {
  modelValue: DataTypeSelection;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  'update:modelValue': [value: DataTypeSelection];
}>();

const localSelection = ref<DataTypeSelection>({ ...props.modelValue });

watch(
  () => props.modelValue,
  (newValue) => {
    localSelection.value = { ...newValue };
  },
  { deep: true }
);

const emitChange = () => {
  emit('update:modelValue', { ...localSelection.value });
};

const selectAll = () => {
  localSelection.value = {
    chats: true,
    prompts: true,
    tools: true,
    files: true,
    models: true,
    knowledge: true,
    users: true,
    groups: true,
    feedbacks: true,
  };
  emitChange();
};

const selectNone = () => {
  localSelection.value = {
    chats: false,
    prompts: false,
    tools: false,
    files: false,
    models: false,
    knowledge: false,
    users: false,
    groups: false,
    feedbacks: false,
  };
  emitChange();
};
</script>

<style scoped>
.data-type-selector {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.data-type-selector h3 {
  margin: 0 0 1rem 0;
  font-size: 1.1rem;
  color: #212529;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.checkbox-label {
  display: flex;
  align-items: flex-start;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.checkbox-label:hover {
  background-color: rgba(0, 0, 0, 0.02);
}

.checkbox-label input[type="checkbox"] {
  margin-right: 0.75rem;
  margin-top: 0.25rem;
  cursor: pointer;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.checkbox-text {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.checkbox-text strong {
  color: #212529;
  font-weight: 600;
}

.checkbox-description {
  color: #6c757d;
  font-size: 0.875rem;
}

.selection-actions {
  display: flex;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid #dee2e6;
}

.btn-secondary {
  padding: 0.5rem 1rem;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: #e9ecef;
}
</style>
