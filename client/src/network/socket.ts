import { Message } from './protocol';

export type MessageHandler = (msg: Message) => void;
export type CloseHandler = () => void;

export class GameSocket {
  private ws: WebSocket | null = null;
  private baseUrl: string;
  private nickname: string;
  private handler: MessageHandler | null = null;
  private closeHandler: CloseHandler | null = null;
  private reconnectTimer: number | null = null;
  private autoReconnect: boolean = true;
  private reconnectAttempts: number = 0;

  private static readonly MAX_RECONNECT_ATTEMPTS = 8;
  private static readonly BASE_DELAY_MS = 1000;
  private static readonly MAX_DELAY_MS = 30000;

  constructor(baseUrl: string, nickname: string) {
    this.baseUrl = baseUrl;
    this.nickname = nickname;
  }

  connect(): void {
    this.autoReconnect = true;
    const url = `${this.baseUrl}?name=${encodeURIComponent(this.nickname)}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0; // reset on successful connection
      if (this.reconnectTimer !== null) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: Message = JSON.parse(event.data);
        if (this.handler) {
          this.handler(msg);
        }
      } catch (e) {
        console.error('Failed to parse message:', e);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket closed');
      if (this.closeHandler) {
        this.closeHandler();
      }
      if (this.autoReconnect) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = (err) => {
      console.error('WebSocket error:', err);
    };
  }

  onMessage(handler: MessageHandler): void {
    this.handler = handler;
  }

  onClose(handler: CloseHandler): void {
    this.closeHandler = handler;
  }

  send(msg: Message): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer !== null) return;
    if (this.reconnectAttempts >= GameSocket.MAX_RECONNECT_ATTEMPTS) {
      console.log('Max reconnect attempts reached, giving up');
      return;
    }

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s, 30s, 30s
    const delay = Math.min(
      GameSocket.BASE_DELAY_MS * Math.pow(2, this.reconnectAttempts),
      GameSocket.MAX_DELAY_MS
    );
    // Add jitter: +/- 25%
    const jitter = delay * (0.75 + Math.random() * 0.5);

    this.reconnectAttempts++;
    console.log(`Reconnecting in ${Math.round(jitter)}ms (attempt ${this.reconnectAttempts}/${GameSocket.MAX_RECONNECT_ATTEMPTS})`);

    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, jitter);
  }

  disconnect(): void {
    this.autoReconnect = false;
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
