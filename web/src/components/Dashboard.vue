<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <h1>Open WebUI Backup Dashboard</h1>
      <button @click="openSettings" class="btn-settings" title="Settings">
        <Settings :size="24" />
      </button>
    </header>

    <div class="dashboard-content">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import {Settings} from 'lucide-vue-next';

const emit = defineEmits<{
  'open-settings': [];
}>();

const openSettings = () => {
  emit('open-settings');
};
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
  background: #f5f7fa;
}

.dashboard-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 2rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dashboard-header h1 {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 600;
}

.btn-settings {
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
  padding: 0.75rem;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-settings:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-1px);
}

.btn-settings:active {
  transform: translateY(0);
}

.dashboard-content {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
}

@media (max-width: 768px) {
  .dashboard-header h1 {
    font-size: 1.5rem;
  }

  .dashboard-content {
    padding: 1rem;
  }
}
</style>
