<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="isOpen" class="modal-backdrop" @click="handleBackdropClick">
        <div class="modal-container" @click.stop>
          <div class="modal-header">
            <div class="header-title">
              <Key :size="24" />
              <h2>Generate Age Identity</h2>
            </div>
            <button @click="closeModal" class="btn-close" aria-label="Close modal">
              <X :size="24" />
            </button>
          </div>
          
          <div class="modal-content">
            <div class="info-section">
              <p>Generate a new age encryption identity pair for secure backup encryption.</p>
              <p class="info-note">
                The <strong>private key (identity)</strong> is used to decrypt backups.
                The <strong>public key (recipient)</strong> is used to encrypt backups.
              </p>
            </div>

            <div v-if="error" class="alert alert-error">
              <AlertCircle :size="20" />
              <span>{{ error }}</span>
            </div>

            <div v-if="!generated" class="generate-section">
              <button 
                @click="handleGenerate" 
                :disabled="loading"
                class="btn btn-primary btn-generate"
              >
                <Key :size="20" v-if="!loading" />
                <Loader2 :size="20" v-if="loading" class="spinning" />
                {{ loading ? 'Generating...' : 'Generate New Identity' }}
              </button>
            </div>

            <div v-if="generated" class="keys-section">
              <div class="key-group">
                <label class="key-label">
                  Private Key (Identity)
                  <span class="key-badge">Keep Secret</span>
                </label>
                <div class="key-display">
                  <textarea 
                    v-model="identity" 
                    readonly 
                    class="key-textarea"
                    rows="3"
                    placeholder="AGE-SECRET-KEY-..."
                  ></textarea>
                  <button 
                    @click="handleCopy(identity, 'identity')" 
                    class="btn btn-icon"
                    :class="{ 'btn-success': copiedIdentity }"
                    title="Copy to clipboard"
                  >
                    <Check :size="20" v-if="copiedIdentity" />
                    <Copy :size="20" v-else />
                  </button>
                </div>
                <p class="key-help">Store this securely. You'll need it to decrypt your backups.</p>
              </div>

              <div class="key-group">
                <label class="key-label">
                  Public Key (Recipient)
                  <span class="key-badge key-badge-public">Safe to Share</span>
                </label>
                <div class="key-display">
                  <textarea 
                    v-model="recipient" 
                    readonly 
                    class="key-textarea"
                    rows="2"
                    placeholder="age1..."
                  ></textarea>
                  <button 
                    @click="handleCopy(recipient, 'recipient')" 
                    class="btn btn-icon"
                    :class="{ 'btn-success': copiedRecipient }"
                    title="Copy to clipboard"
                  >
                    <Check :size="20" v-if="copiedRecipient" />
                    <Copy :size="20" v-else />
                  </button>
                </div>
                <p class="key-help">Use this to encrypt backups. You can share this publicly.</p>
              </div>

              <div class="actions-section">
                <button 
                  @click="handleSave" 
                  class="btn btn-primary"
                >
                  <Save :size="20" />
                  Save to Current Configuration
                </button>
                <button 
                  @click="handleGenerate" 
                  class="btn btn-secondary"
                  :disabled="loading"
                >
                  <RefreshCw :size="20" />
                  Generate Another
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {onMounted, onUnmounted, ref, watch} from 'vue';
import {AlertCircle, Check, Copy, Key, Loader2, RefreshCw, Save, X} from 'lucide-vue-next';
import {generateIdentity} from '../services/api';

interface Props {
  isOpen: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'close': [];
  'save-identity': [identity: string, recipient: string];
}>();

const loading = ref(false);
const error = ref('');
const identity = ref('');
const recipient = ref('');
const generated = ref(false);
const copiedIdentity = ref(false);
const copiedRecipient = ref(false);

const handleGenerate = async () => {
  loading.value = true;
  error.value = '';
  
  try {
    const response = await generateIdentity();
    identity.value = response.identity;
    recipient.value = response.recipient;
    generated.value = true;
    copiedIdentity.value = false;
    copiedRecipient.value = false;
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to generate identity';
    console.error('Failed to generate identity:', err);
  } finally {
    loading.value = false;
  }
};

const handleSave = () => {
  if (identity.value && recipient.value) {
    emit('save-identity', identity.value, recipient.value);
    closeModal();
  }
};

const handleCopy = async (text: string, type: 'identity' | 'recipient') => {
  try {
    await navigator.clipboard.writeText(text);
    if (type === 'identity') {
      copiedIdentity.value = true;
      setTimeout(() => {
        copiedIdentity.value = false;
      }, 2000);
    } else {
      copiedRecipient.value = true;
      setTimeout(() => {
        copiedRecipient.value = false;
      }, 2000);
    }
  } catch (err) {
    console.error('Failed to copy to clipboard:', err);
    error.value = 'Failed to copy to clipboard';
  }
};

const closeModal = () => {
  // Reset state when closing
  loading.value = false;
  error.value = '';
  identity.value = '';
  recipient.value = '';
  generated.value = false;
  copiedIdentity.value = false;
  copiedRecipient.value = false;
  emit('close');
};

const handleBackdropClick = () => {
  closeModal();
};

const handleEscape = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.isOpen) {
    closeModal();
  }
};

// Handle ESC key and body scroll
watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    document.addEventListener('keydown', handleEscape);
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
  max-width: 700px;
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

.header-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: #212529;
}

.header-title h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
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

.info-section {
  margin-bottom: 1.5rem;
}

.info-section p {
  margin: 0.5rem 0;
  color: #495057;
  line-height: 1.6;
}

.info-note {
  background: #e7f3ff;
  padding: 1rem;
  border-radius: 8px;
  border-left: 4px solid #007bff;
  font-size: 0.9rem;
}

.alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1rem;
}

.alert-error {
  background: #fee;
  color: #c00;
  border-left: 4px solid #c00;
}

.generate-section {
  display: flex;
  justify-content: center;
  padding: 2rem 0;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  text-decoration: none;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background: #5a6268;
}

.btn-generate {
  font-size: 1.1rem;
  padding: 1rem 2rem;
}

.btn-icon {
  padding: 0.5rem;
  background: #f8f9fa;
  color: #495057;
}

.btn-icon:hover:not(:disabled) {
  background: #e9ecef;
}

.btn-success {
  background: #28a745 !important;
  color: white !important;
}

.keys-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.key-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.key-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #212529;
  font-size: 0.95rem;
}

.key-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: #ffc107;
  color: #000;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.key-badge-public {
  background: #28a745;
  color: white;
}

.key-display {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
}

.key-textarea {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #ced4da;
  border-radius: 6px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  resize: none;
  background: #f8f9fa;
  color: #212529;
}

.key-textarea:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 0.2rem rgba(0, 123, 255, 0.25);
}

.key-help {
  margin: 0;
  font-size: 0.85rem;
  color: #6c757d;
  font-style: italic;
}

.actions-section {
  display: flex;
  gap: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #e9ecef;
}

.actions-section .btn {
  flex: 1;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
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

  .actions-section {
    flex-direction: column;
  }

  .key-display {
    flex-direction: column;
  }

  .btn-icon {
    align-self: flex-start;
  }
}
</style>
