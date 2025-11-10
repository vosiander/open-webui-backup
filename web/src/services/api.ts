import type {
    BackupRequest,
    ConfigResponse,
    GenerateIdentityResponse,
    OperationStartResponse,
    OperationStatus,
    RestoreRequest,
    UpdateConfigRequest,
} from '../types/api';

const API_BASE = '/api';

export class APIError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public details?: unknown
  ) {
    super(message);
    this.name = 'APIError';
  }
}

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (!response.ok) {
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
      try {
        const errorData = await response.json();
        if (errorData.error) {
          errorMessage = errorData.error;
        }
      } catch {
        // Ignore JSON parse errors for error responses
      }
      throw new APIError(errorMessage, response.status);
    }

    return await response.json();
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }
    throw new APIError(
      error instanceof Error ? error.message : 'Network error',
      0
    );
  }
}

export async function fetchConfig(): Promise<ConfigResponse> {
  return fetchJSON<ConfigResponse>(`${API_BASE}/config`);
}

export async function updateConfig(
  request: UpdateConfigRequest
): Promise<ConfigResponse> {
  return fetchJSON<ConfigResponse>(`${API_BASE}/config`, {
    method: 'PUT',
    body: JSON.stringify(request),
  });
}

export async function startBackup(
  request: BackupRequest
): Promise<OperationStartResponse> {
  return fetchJSON<OperationStartResponse>(`${API_BASE}/backup`, {
    method: 'POST',
    body: JSON.stringify(request),
  });
}

export async function startRestore(
  request: RestoreRequest
): Promise<OperationStartResponse> {
  return fetchJSON<OperationStartResponse>(`${API_BASE}/restore`, {
    method: 'POST',
    body: JSON.stringify(request),
  });
}

export async function getOperationStatus(
  operationId: string
): Promise<OperationStatus> {
  return fetchJSON<OperationStatus>(`${API_BASE}/status/${operationId}`);
}

export interface BackupFile {
  name: string;
  size: number;
  modTime: string;
  downloadUrl: string;
}

export async function listBackups(): Promise<BackupFile[]> {
  return fetchJSON<BackupFile[]>(`${API_BASE}/backups`);
}

export function getDownloadUrl(filename: string): string {
  return `${API_BASE}/backups/${encodeURIComponent(filename)}`;
}

export async function deleteBackup(filename: string): Promise<void> {
  await fetchJSON(`${API_BASE}/backups/${encodeURIComponent(filename)}`, {
    method: 'DELETE',
  });
}

export async function uploadBackup(file: File): Promise<{ filename: string }> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch(`${API_BASE}/backups/upload`, {
    method: 'POST',
    body: formData,
    // Don't set Content-Type header - let browser set it with boundary
  });

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
    try {
      const errorData = await response.json();
      if (errorData.error) {
        errorMessage = errorData.error;
      }
    } catch {
      // Ignore JSON parse errors for error responses
    }
    throw new APIError(errorMessage, response.status);
  }

  return await response.json();
}

export async function verifyBackup(
  filename: string,
  identity: string
): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/backups/verify`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      filename,
      decryptIdentity: identity,
    }),
  });

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
    try {
      const errorData = await response.json();
      if (errorData.error) {
        errorMessage = errorData.error;
      }
    } catch {
      // Ignore JSON parse errors for error responses
    }
    throw new APIError(errorMessage, response.status);
  }

  const data = await response.json();
  return {
    success: data.success === 'true',
    message: data.message || '',
  };
}

export async function generateIdentity(): Promise<GenerateIdentityResponse> {
  return fetchJSON<GenerateIdentityResponse>(`${API_BASE}/identity/generate`, {
    method: 'POST',
  });
}
