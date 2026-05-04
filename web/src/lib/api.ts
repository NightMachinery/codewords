import type { LobbyPlayer } from './lobby';
import type { ClueEntry, GameplayCard, LastSelected, RemainingCounts } from './gameplay';

export interface ApiErrorBody {
  error?: { code?: string; message?: string };
}

export interface Settings {
  seed: number;
  blackCards: number;
  wordpackId: string;
  enforceClueGuessLimit: boolean;
  allowInfinityClue: boolean;
  imageCardCount: number;
  randomizeTeams: boolean;
  customColorBlue?: string;
  customColorRed?: string;
  observerChatEnabled: boolean;
  mixedImageOrderFirst: boolean;
}

export interface Viewer {
  userId: string;
  playerId?: string;
  isHost: boolean;
  isMod?: boolean;
}

export interface RoomSummary {
  id: string;
  hostUserId: string;
  status: string;
  currentMatchId: string;
}

export interface RoomSnapshot {
  phase: 'lobby' | 'active' | 'game_over';
  players: LobbyPlayer[];
  settings: Settings;
  currentTeam: 'blue' | 'red' | '';
  winner: 'blue' | 'red' | '';
  actionId: number;
  cards: GameplayCard[];
  lastSelected?: LastSelected | null;
  remainingCounts: RemainingCounts;
  clueLog: ClueEntry[];
  viewer: Viewer;
}

export interface Wordpack {
  id: string;
  label: string;
  wordCount: number;
}

export interface PictureAsset {
  id: string;
  url: string;
}

export interface ChatMessage {
  id: string;
  roomId: string;
  matchId: string;
  senderUserId: string;
  displayName: string;
  body: string;
  createdAt: string;
}

export interface Credential {
  authToken?: string;
  migrateId?: string;
}

export class ApiClient {
  async bootstrap(authToken: string, displayName = ''): Promise<{ userId: string; displayName: string }> {
    return this.post('/api/identity/bootstrap', { authToken, displayName });
  }

  async saveDisplayName(authToken: string, displayName: string): Promise<{ userId: string; displayName: string }> {
    return this.post('/api/identity/display-name', { authToken, displayName });
  }

  async createRoom(authToken: string, settings: Settings): Promise<{ room: RoomSummary; roomLink: string; viewer: Viewer }> {
    return this.post('/api/rooms', { authToken, settings });
  }

  async getRoom(
    roomId: string,
    credential: Credential,
  ): Promise<{ room: RoomSummary; players: LobbyPlayer[]; settings: Settings; viewer: Viewer; chatMessages?: ChatMessage[] }> {
    const params = credential.migrateId ? `migrateId=${encodeURIComponent(credential.migrateId)}` : `authToken=${encodeURIComponent(credential.authToken ?? '')}`;
    return this.get(`/api/rooms/${encodeURIComponent(roomId)}?${params}`);
  }

  async joinRoom(roomId: string, authToken: string, displayName: string): Promise<{ room: RoomSummary; viewer: Viewer }> {
    return this.post(`/api/rooms/${encodeURIComponent(roomId)}/join`, { authToken, displayName });
  }

  async updateSettings(roomId: string, authToken: string, settings: Settings): Promise<{ settings: Settings }> {
    return this.post(`/api/rooms/${encodeURIComponent(roomId)}/settings`, { authToken, settings });
  }

  async startRoom(roomId: string, authToken: string): Promise<{ matchId: string; snapshot: RoomSnapshot }> {
    return this.post(`/api/rooms/${encodeURIComponent(roomId)}/start`, { authToken });
  }

  async createMigrateLink(roomId: string, authToken: string): Promise<{ migrateUrl: string; migrateId: string }> {
    return this.post(`/api/rooms/${encodeURIComponent(roomId)}/migrate-link`, { authToken });
  }

  async migrateBootstrap(roomId: string, migrateId: string): Promise<{ roomId: string; userId: string; displayName: string }> {
    return this.post(`/api/rooms/${encodeURIComponent(roomId)}/migrate-bootstrap`, { migrateId });
  }

  async wordpacks(): Promise<{ wordpacks: Wordpack[] }> {
    return this.get('/api/wordpacks');
  }

  async pictureCatalog(): Promise<{ available: boolean; images: PictureAsset[] }> {
    return this.get('/api/pictures/catalog');
  }

  private async get<T>(path: string): Promise<T> {
    return this.request(path, { method: 'GET' });
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    return this.request(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  }

  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const response = await fetch(path, init);
    const payload = (await response.json().catch(() => ({}))) as T & ApiErrorBody;
    if (!response.ok) {
      throw new Error(payload.error?.message ?? `Request failed with ${response.status}`);
    }
    return payload;
  }
}

export const defaultSettings: Settings = {
  seed: Date.now(),
  blackCards: 1,
  wordpackId: 'english',
  enforceClueGuessLimit: false,
  allowInfinityClue: false,
  imageCardCount: 0,
  randomizeTeams: true,
  observerChatEnabled: true,
  mixedImageOrderFirst: false,
};

export const api = new ApiClient();
