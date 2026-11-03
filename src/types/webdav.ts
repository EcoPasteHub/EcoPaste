export interface WebDavSyncResult {
  success: boolean;
  uploadedItems: number;
  uploadedImages: number;
  backedUpSettings: boolean;
  error: string | null;
  syncedAt: string;
}

export interface WebDavRestoreResult {
  success: boolean;
  restoredItems: number;
  restoredImages: number;
  restoredSettings: boolean;
  error: string | null;
}

export interface WebDavTestResult {
  success: boolean;
  message: string;
}

