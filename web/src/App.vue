<template>
  <Dashboard @open-settings="openConfigModal">
    <div class="current-operation">
      <OperationProgress 
        :status="currentOperation"
        :formError="formError"
        :formSuccess="formSuccess"
      />
    </div>

    <div class="operations-grid">
      <div class="operation-section">
        <BackupForm 
          @operation-started="handleOperationStarted"
          @operation-error="handleOperationError"
        />
      </div>

      <div class="operation-section">
        <RestoreForm 
          :ageIdentity="ageIdentity"
          @operation-started="handleOperationStarted"
          @operation-error="handleOperationError"
        />
      </div>
    </div>

    <BackupList ref="backupListRef" />
  </Dashboard>

  <ConfigModal 
    :isOpen="isConfigModalOpen"
    @close="closeConfigModal"
    @update:ageIdentity="handleAgeIdentityUpdate"
    @update:ageRecipients="handleAgeRecipientsUpdate"
  />
</template>

<script setup lang="ts">
import {onMounted, ref} from 'vue';
import Dashboard from './components/Dashboard.vue';
import ConfigModal from './components/ConfigModal.vue';
import BackupForm from './components/BackupForm.vue';
import RestoreForm from './components/RestoreForm.vue';
import OperationProgress from './components/OperationProgress.vue';
import BackupList from './components/BackupList.vue';
import {useWebSocket} from './composables/useWebSocket';
import type {OperationStatus} from './types/api';

const { addMessageHandler } = useWebSocket();
const currentOperation = ref<OperationStatus | null>(null);
const backupListRef = ref<InstanceType<typeof BackupList> | null>(null);
const formError = ref<string | null>(null);
const formSuccess = ref<{ operationId: string; type: string } | null>(null);
const isConfigModalOpen = ref(false);
const ageIdentity = ref('');
const ageRecipients = ref('');

const openConfigModal = () => {
  isConfigModalOpen.value = true;
};

const closeConfigModal = () => {
  isConfigModalOpen.value = false;
};

const handleAgeIdentityUpdate = (value: string) => {
  ageIdentity.value = value;
};

const handleAgeRecipientsUpdate = (value: string) => {
  ageRecipients.value = value;
};

const handleOperationStarted = (payload: { operationId: string; type: string }) => {
  console.log('Operation started:', payload);
  
  // Clear any previous errors
  formError.value = null;
  
  // Show success message
  formSuccess.value = payload;
  
  // Clear success message after delay
  setTimeout(() => {
    formSuccess.value = null;
  }, 5000);
};

const handleOperationError = (payload: { message: string; type: string }) => {
  console.log('Operation error:', payload);
  
  // Clear any previous success
  formSuccess.value = null;
  
  // Show error message
  formError.value = payload.message;
  
  // Clear error message after delay
  setTimeout(() => {
    formError.value = null;
  }, 8000);
};

onMounted(() => {
  // Listen for WebSocket messages
  addMessageHandler((message) => {
    if (message.type === 'status') {
      const status = message.payload as OperationStatus;
      currentOperation.value = status;

      // Clear form messages when WebSocket status arrives
      if (status.status === 'running') {
        formError.value = null;
        formSuccess.value = null;
      }

      // Refresh backup list when backup completes
      if (status.type === 'backup' && status.status === 'completed') {
        setTimeout(() => {
          backupListRef.value?.refresh();
        }, 1000);
      }

      // Clear completed/failed operations after a delay
      if (status.status === 'completed' || status.status === 'failed') {
        setTimeout(() => {
          if (currentOperation.value?.id === status.id) {
            currentOperation.value = null;
          }
        }, 10000);
      }
    }
  });
});
</script>

<style scoped>
.operations-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

.operation-section {
  min-width: 0;
  display: block;
  visibility: visible;
}

.current-operation {
  margin-bottom: 1.5rem;
}

@media (max-width: 1024px) {
  .operations-grid {
    grid-template-columns: 1fr;
    gap: 1.25rem;
  }
}
</style>
