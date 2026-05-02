import type { ChatMessage, RoomSnapshot } from './api';

export type RoomSocketMessage =
  | { type: 'snapshot'; snapshot: RoomSnapshot }
  | { type: 'error'; code: string; message: string }
  | { type: 'chatMessage'; message: ChatMessage }
  | { type: 'pong' };

export interface RoomSocketHandlers {
  onMessage(message: RoomSocketMessage): void;
  onStatus(status: 'connecting' | 'connected' | 'disconnected'): void;
}

export class RoomSocket {
  private socket: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;
  private attempts = 0;

  constructor(
    private readonly url: string,
    private readonly handlers: RoomSocketHandlers,
  ) {}

  connect(): void {
    this.closed = false;
    this.handlers.onStatus('connecting');
    this.socket = new WebSocket(this.url);
    this.socket.onopen = () => {
      this.attempts = 0;
      this.handlers.onStatus('connected');
    };
    this.socket.onmessage = (event) => {
      this.handlers.onMessage(JSON.parse(event.data) as RoomSocketMessage);
    };
    this.socket.onclose = () => {
      this.handlers.onStatus('disconnected');
      this.scheduleReconnect();
    };
    this.socket.onerror = () => {
      this.socket?.close();
    };
  }

  send(message: Record<string, unknown>): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  close(): void {
    this.closed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.socket?.close();
    this.socket = null;
  }

  private scheduleReconnect(): void {
    if (this.closed) {
      return;
    }
    const delay = Math.min(5000, 400 * 2 ** this.attempts);
    this.attempts += 1;
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }
}
