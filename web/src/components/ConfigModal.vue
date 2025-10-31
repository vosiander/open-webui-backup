<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="isOpen" class="modal-backdrop" @click="handleBackdropClick">
        <div class="modal-container" @click.stop>
          <div class="modal-header">
            <h2>Settings</h2>
            <button @click="closeModal" class="btn-close" aria-label="Close modal">
              <X :size="24" />
            </button>
          </div>
          <div class="modal-content">
            <ConfigPanel
              @update:ageIdentity="handleAgeIdentityUpdate"
              @update:ageRecipients="handleAgeRecipientsUpdate"
            />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {onMounted, onUnmounted, watch} from 'vue';
import {X} from 'lucide-vue-next';
import ConfigPanel from './ConfigPanel.vue';

interface Props {
  isOpen: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'close': [];
  'update:ageIdentity': [value: string];
  'update:ageRecipients': [value: string];
}>();

const closeModal = () => {
  emit('close');
};

const handleBackdropClick = () => {
  closeModal();
};

const handleAgeIdentityUpdate = (value: string) => {
  emit('update:ageIdentity', value);
};

const handleAgeRecipientsUpdate = (value: string) => {
  emit('update:ageRecipients', value);
};

const handleEscape = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.isOpen) {
    closeModal();
  }
};

// Handle ESC key
watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    document.addEventListener('keydown', handleEscape);
    // Prevent body scroll when modal is open
    document.body.style.overflow = 'hidden';
  } else {
    document.removeEventListener('keydown', handleEscape);
    document.body.style.overflow = '';
  }
});

onMounted(() => {
  if (props.isOpen) {
    document.addEventListener('keydown', handleEscape);
  }
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape);
  document.body.style.overflow = '';
});
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-container {
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  max-width: 900px;
  width: 100%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e9ecef;
  background: #f8f9fa;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #212529;
}

.btn-close {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
  color: #6c757d;
}

.btn-close:hover {
  background: #e9ecef;
  color: #212529;
}

.modal-content {
  padding: 1.5rem;
  overflow-y: auto;
}

/* Remove extra padding from ConfigPanel when in modal */
.modal-content :deep(.config-panel) {
  padding: 0;
  margin: 0;
  box-shadow: none;
  border: none;
}

/* Animations */
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-active .modal-container,
.modal-fade-leave-active .modal-container {
  transition: transform 0.2s ease;
}

.modal-fade-enter-from .modal-container,
.modal-fade-leave-to .modal-container {
  transform: scale(0.95);
}

@media (max-width: 768px) {
  .modal-container {
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .modal-backdrop {
    padding: 0;
  }
}
</style>
