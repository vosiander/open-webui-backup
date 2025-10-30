<template>
  <div class="backup-list">
    <div class="list-header">
      <h3>Available Backups</h3>
      <button @click="refreshList" class="btn-refresh" :disabled="loading">
        <span v-if="loading">Refreshing...</span>
        <span v-else>Refresh</span>
      </button>
    </div>

    <div v-if="loading && backups.length === 0" class="loading">
      Loading backups...
    </div>

    <div v-else-if="error" class="error">
      {{ error }}
    </div>

    <div v-else-if="backups.length === 0" class="empty-state">
      <p>No backups found</p>
      <p class="empty-hint">Create a backup to get started</p>
    </div>

    <div v-else class="backup-items">
      <div
        v-for="backup in sortedBackups"
        :key="backup.name"
        class="backup-item"
      >
        <div class="backup-info">
          <div class="backup-name">{{ backup.name }}</div>
          <div class="backup-meta">
            <span class="backup-size">{{ formatSize(backup.size) }}</span>
            <span class="backup-date">{{ formatDate(backup.modTime) }}</span>
          </div>
        </div>
        <div class="backup-actions">
          <a
            :href="backup.downloadUrl"
            class="btn-download"
            download
            title="Download backup"
          >
            Download
          </a>
          <button
            @click="handleDelete(backup)"
            class="btn-delete"
            title="Delete backup"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue';
import {type BackupFile, deleteBackup, listBackups} from '../services/api';

const backups = ref<BackupFile[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const sortedBackups = computed(() => {
  return [...backups.value].sort((a, b) => {
    return new Date(b.modTime).getTime() - new Date(a.modTime).getTime();
  });
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
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) {
    return 'Just now';
  } else if (diffMins < 60) {
    return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`;
  } else if (diffHours < 24) {
    return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
  } else if (diffDays < 7) {
    return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
  } else {
    return date.toLocaleDateString();
  }
};

const handleDelete = async (backup: BackupFile) => {
  if (!confirm(`Are you sure you want to delete "${backup.name}"? This action cannot be undone.`)) {
    return;
  }

  try {
    await deleteBackup(backup.name);
    await refreshList();
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete backup';
  }
};

const refreshList = async () => {
  loading.value = true;
  error.value = null;

  try {
    backups.value = await listBackups();
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load backups';
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  refreshList();
});

// Expose refresh method to parent
defineExpose({
  refresh: refreshList,
});
</script>

<style scoped>
.backup-list {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.list-header h3 {
  margin: 0;
  font-size: 1.25rem;
  color: #212529;
}

.btn-refresh {
  padding: 0.5rem 1rem;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.btn-refresh:hover:not(:disabled) {
  background: #f8f9fa;
}

.btn-refresh:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading,
.error,
.empty-state {
  padding: 2rem;
  text-align: center;
  color: #6c757d;
}

.error {
  color: #dc3545;
  background: #f8d7da;
  border-radius: 4px;
}

.empty-state p {
  margin: 0.5rem 0;
}

.empty-hint {
  font-size: 0.875rem;
  color: #6c757d;
}

.backup-items {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  transition: all 0.2s;
}

.backup-item:hover {
  border-color: #667eea;
  box-shadow: 0 2px 4px rgba(102, 126, 234, 0.1);
}

.backup-info {
  flex: 1;
  min-width: 0;
}

.backup-name {
  font-weight: 600;
  color: #212529;
  margin-bottom: 0.25rem;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.9375rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.backup-meta {
  display: flex;
  gap: 1rem;
  font-size: 0.875rem;
  color: #6c757d;
}

.backup-actions {
  display: flex;
  gap: 0.5rem;
  margin-left: 1rem;
}

.btn-download {
  padding: 0.5rem 1rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 4px;
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: inline-block;
}

.btn-download:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.btn-delete {
  padding: 0.5rem 1rem;
  background: white;
  color: #dc3545;
  border: 1px solid #dc3545;
  border-radius: 4px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-delete:hover {
  background: #dc3545;
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(220, 53, 69, 0.3);
}

@media (max-width: 768px) {
  .backup-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .backup-actions {
    margin-left: 0;
    width: 100%;
  }

  .btn-download {
    width: 100%;
    text-align: center;
  }
}
</style>
