export interface BackupRequest {
  outputFilename: string;
  encryptRecipients: string[];
  dataTypes: DataTypeSelection;
}

export interface RestoreRequest {
  inputFilename: string;
  decryptIdentity: string;
  dataTypes: DataTypeSelection;
  overwrite: boolean;
}

export interface DataTypeSelection {
  prompts: boolean;
  tools: boolean;
  knowledge: boolean;
  models: boolean;
  files: boolean;
  chats: boolean;
  users: boolean;
  groups: boolean;
  feedbacks: boolean;
}

export interface OperationStatus {
  id: string;
  type: 'backup' | 'restore';
  status: 'running' | 'completed' | 'failed';
  progress: number;
  message: string;
  startTime: string;
  endTime?: string;
  error?: string;
  outputFile?: string;
}

export interface ConfigResponse {
  openWebUIURL: string;
  apiKey?: string;
  serverPort: number;
  backupsDir: string;
  defaultRecipient: string;
  defaultIdentity: string;
  defaultAgeIdentity?: string;
  defaultAgeRecipients?: string;
  availableBackups: string[];
}

export interface UpdateConfigRequest {
  openWebUIURL?: string;
  apiKey?: string;
}

export interface WebSocketMessage {
  type: 'status' | 'progress' | 'log';
  payload: any;
}

export interface OperationStartResponse {
  operationId: string;
}

export interface GenerateIdentityResponse {
  identity: string;
  recipient: string;
}
