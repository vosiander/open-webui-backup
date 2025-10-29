<template>
  <Dashboard>
    <ConfigPanel
      @update:ageIdentity="handleAgeIdentityUpdate"
      @update:ageRecipients="handleAgeRecipientsUpdate"
    />

    <div class="current-operation">
      <OperationProgress :status="currentOperation" />
    </div>

    <div class="operations-grid">
      <div class="operation-section">
        <BackupForm @backup-started="handleBackupStarted" />
      </div>

      <div class="operation-section">
        <RestoreForm @restore-started="handleRestoreStarted" />
      </div>
    </div>

    <BackupList ref="backupListRef" />
  </Dashboard>
</template>

<script setup lang="ts">
import {onMounted, ref} from 'vue';
import Dashboard from './components/Dashboard.vue';
import ConfigPanel from './components/ConfigPanel.vue';
import BackupForm from './components/BackupForm.vue';
import RestoreForm from './components/RestoreForm.vue';
import OperationProgress from './components/OperationProgress.vue';
import BackupList from './components/BackupList.vue';
import {useWebSocket} from './composables/useWebSocket';
import type {OperationStatus} from './types/api';

const { addMessageHandler } = useWebSocket();
const currentOperation = ref<OperationStatus | null>(null);
const backupListRef = ref<InstanceType<typeof BackupList> | null>(null);
const ageIdentity = ref('');
const ageRecipients = ref('');

const handleAgeIdentityUpdate = (value: string) => {
  ageIdentity.value = value;
};

const handleAgeRecipientsUpdate = (value: string) => {
  ageRecipients.value = value;
};

const handleBackupStarted = (operationId: string) => {
  console.log('Backup started:', operationId);
  // Current operation will be updated via WebSocket
};

const handleRestoreStarted = (operationId: string) => {
  console.log('Restore started:', operationId);
  // Current operation will be updated via WebSocket
};

onMounted(() => {
  // Listen for WebSocket messages
  addMessageHandler((message) => {
    if (message.type === 'status') {
      const status = message.payload as OperationStatus;
      currentOperation.value = status;

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
