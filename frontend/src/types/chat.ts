export interface ChatMessage {
  id: string;
  room_id: string;
  user_id: string;
  username: string;
  message: string;
  timestamp: string;
}

export interface UserLogEntry {
  id: string;
  room_id: string;
  user_id: string;
  username: string;
  action: string;
  timestamp: string;
  data?: {
    current_time?: number;
    duration?: number;
    playback_rate?: number;
    is_buffering?: boolean;
  };
}

export interface UserLogsResponse {
  logs: UserLogEntry[];
}
