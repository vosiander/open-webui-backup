<template>
  <div class="operation-progress">
    <div class="progress-header">
      <h3>{{ status ? (status.type === 'backup' ? 'Backup' : 'Restore') + ' Progress' : 'Operation Status' }}</h3>
      <span v-if="status" class="operation-id">ID: {{ status.id }}</span>
    </div>

    <div class="status-badge" :class="statusClass">
      {{ statusText }}
    </div>

    <div v-if="status" class="progress-bar-container">
      <div class="progress-bar" :style="{ width: `${status.progress ?? 0}%` }">
        <span class="progress-text">{{ status.progress ?? 0 }}%</span>
      </div>
    </div>

    <div v-if="formError" class="alert alert-error">
      <strong>Error:</strong> {{ formError }}
    </div>

    <div v-if="formSuccess" class="alert alert-success">
      {{ formSuccess.type === 'backup' ? 'Backup' : 'Restore' }} started successfully! Operation ID: {{ formSuccess.operationId }}
    </div>

    <div v-if="!status && !formError && !formSuccess" class="ready-message">
      No operations currently running. Start a backup or restore to see progress here.
    </div>

    <div class="progress-message" v-if="status && status.message">
      {{ status.message }}
    </div>

    <div class="operation-details" v-if="status && status.startTime">
      <div class="detail-item">
        <strong>Started:</strong> {{ formatDate(status.startTime) }}
      </div>
      <div class="detail-item" v-if="status.endTime">
        <strong>Ended:</strong> {{ formatDate(status.endTime) }}
      </div>
      <div class="detail-item" v-if="status.outputFile">
        <strong>Output File:</strong>
        <a :href="getDownloadUrl(status.outputFile)" class="download-link" download>
          {{ status.outputFile }}
        </a>
      </div>
    </div>

    <div class="error-message" v-if="status && status.error">
      <strong>Error:</strong> {{ status.error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import {computed} from 'vue';
import type {OperationStatus} from '../types/api';
import {getDownloadUrl} from '../services/api';

interface Props {
  status: OperationStatus | null;
  formError: string | null;
  formSuccess: { operationId: string; type: string } | null;
}

const props = defineProps<Props>();

const statusClass = computed(() => {
  if (!props.status) return '';
  switch (props.status.status) {
    case 'running':
      return 'status-running';
    case 'completed':
      return 'status-completed';
    case 'failed':
      return 'status-failed';
    default:
      return '';
  }
});

const statusText = computed(() => {
  if (!props.status) return 'Ready';
  switch (props.status.status) {
    case 'running':
      return 'In Progress';
    case 'completed':
      return 'Completed';
    case 'failed':
      return 'Failed';
    default:
      return props.status.status;
  }
});

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr);
  return date.toLocaleString();
};
</script>

<style scoped>
.operation-progress {
  background: white;
  border-radius: 8px;
  padding: 1.25rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  border: 1px solid #e9ecef;
  margin-bottom: 1.5rem;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.progress-header h3 {
  margin: 0;
  font-size: 1.125rem;
  color: #212529;
  font-weight: 600;
}

.operation-id {
  font-size: 0.75rem;
  color: #6c757d;
  font-family: 'Monaco', 'Courier New', monospace;
  background: #f8f9fa;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.status-badge {
  display: inline-block;
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
}

.status-running {
  background: #fff3cd;
  color: #856404;
  border: 1px solid #ffeeba;
  animation: pulse 2s ease-in-out infinite;
}

.status-completed {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.status-failed {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.ready-message {
  color: #6c757d;
  font-size: 0.875rem;
  padding: 1rem;
  text-align: center;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

.progress-bar-container {
  width: 100%;
  height: 28px;
  background: #e9ecef;
  border-radius: 14px;
  overflow: hidden;
  margin-bottom: 0.75rem;
  position: relative;
  border: 1px solid #dee2e6;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  transition: width 0.4s ease-out;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 50px;
  box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.3);
}

.progress-text {
  color: white;
  font-weight: 600;
  font-size: 0.8125rem;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.progress-message {
  color: #495057;
  font-size: 0.875rem;
  margin-bottom: 0.75rem;
  padding: 0.625rem 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  border-left: 3px solid #667eea;
  font-family: 'Monaco', 'Courier New', monospace;
}

.operation-details {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding-top: 0.75rem;
  border-top: 1px solid #e9ecef;
}

.detail-item {
  font-size: 0.8125rem;
  color: #495057;
}

.detail-item strong {
  color: #212529;
  margin-right: 0.5rem;
  font-weight: 600;
}

.download-link {
  color: #667eea;
  text-decoration: none;
  font-family: 'Monaco', 'Courier New', monospace;
  font-weight: 500;
  transition: color 0.15s;
}

.download-link:hover {
  color: #5568d3;
  text-decoration: underline;
}

.error-message {
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: #f8d7da;
  border: 1px solid #f5c6cb;
  border-radius: 6px;
  color: #721c24;
  font-size: 0.8125rem;
}

.error-message strong {
  display: block;
  margin-bottom: 0.375rem;
  font-weight: 600;
}

.alert {
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
}

.alert strong {
  font-weight: 600;
  margin-right: 0.5rem;
}

.alert-error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.alert-success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}
</style>
