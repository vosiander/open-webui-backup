import {computed, ref} from 'vue';
import type {OperationStatus} from '../types/api';
import {getOperationStatus} from '../services/api';

export function useOperation(operationId: string) {
  const status = ref<OperationStatus | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const pollInterval = ref<number | null>(null);

  const isRunning = computed(() => status.value?.status === 'running');
  const isCompleted = computed(() => status.value?.status === 'completed');
  const isFailed = computed(() => status.value?.status === 'failed');

  const fetchStatus = async () => {
    loading.value = true;
    error.value = null;

    try {
      status.value = await getOperationStatus(operationId);
      
      // Stop polling if operation is completed or failed
      if (status.value.status === 'completed' || status.value.status === 'failed') {
        stopPolling();
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch operation status';
      stopPolling();
    } finally {
      loading.value = false;
    }
  };

  const startPolling = (interval: number = 1000) => {
    if (pollInterval.value !== null) {
      return;
    }

    // Fetch immediately
    fetchStatus();

    // Then poll at interval
    pollInterval.value = window.setInterval(() => {
      fetchStatus();
    }, interval);
  };

  const stopPolling = () => {
    if (pollInterval.value !== null) {
      clearInterval(pollInterval.value);
      pollInterval.value = null;
    }
  };

  const updateFromWebSocket = (wsStatus: OperationStatus) => {
    status.value = wsStatus;
    
    // Stop polling if operation is completed or failed
    if (wsStatus.status === 'completed' || wsStatus.status === 'failed') {
      stopPolling();
    }
  };

  return {
    status,
    loading,
    error,
    isRunning,
    isCompleted,
    isFailed,
    fetchStatus,
    startPolling,
    stopPolling,
    updateFromWebSocket,
  };
}
