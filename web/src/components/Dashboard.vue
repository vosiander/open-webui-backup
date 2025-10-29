<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <h1>Open WebUI Backup Dashboard!!!</h1>
    </header>

    <div class="dashboard-content">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
// Connection status removed - operation status is shown in OperationProgress component
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
