import { useState, useEffect } from 'react';
import './UserLogs.css';

interface UserLogEntry {
  id: string;
  username: string;
  action: string;
  timestamp: string;
  data?: {
    current_time?: number;
    duration?: number;
    playback_rate?: number;
    is_buffering?: boolean;
    chat_message?: string;
  };
}

interface SyncEvent {
  id?: string;
  username?: string;
  action: string;
  timestamp?: string;
  data?: {
    current_time?: number;
    duration?: number;
    playback_rate?: number;
    is_buffering?: boolean;
    chat_message?: string;
  };
}

interface UserLogsProps {
  isVisible: boolean;
  isAdmin: boolean; // only admins can see user logs
  syncEvents?: SyncEvent[]; // unified sync events from WebSocket
}

export function UserLogs({ isVisible, isAdmin, syncEvents }: UserLogsProps) {
  const [logs, setLogs] = useState<UserLogEntry[]>([]);

  // process sync events when they change - admin only
  useEffect(() => {
    if (!isAdmin) {
      setLogs([]);
      return;
    }

    console.log('UserLogs: received sync events:', syncEvents);
    
    if (!syncEvents || syncEvents.length === 0) return;

    // show user actions but exclude chat messages (chat has its own component)
    const relevantActions = ['play', 'pause', 'seek', 'join', 'leave', 'buffering', 'ready'];
    
    // process all sync events and convert to log entries
    const newLogEntries: UserLogEntry[] = [];
    
    syncEvents.forEach(event => {
      console.log('UserLogs: processing event:', event);
      
      // check if syncEvent is valid
      if (!event || !event.action) {
        console.warn('received invalid sync event:', event)
        return
      }
      
      console.log('UserLogs: checking if action is relevant:', event.action, 'relevant actions:', relevantActions);
      
      if (relevantActions.includes(event.action)) {
        const logEntry: UserLogEntry = {
          id: event.id || `${Date.now()}-${Math.random()}`,
          username: event.username || 'Unknown User',
          action: event.action,
          timestamp: event.timestamp || new Date().toISOString(),
          data: {
            current_time: event.data?.current_time,
            duration: event.data?.duration,
            playback_rate: event.data?.playback_rate,
            is_buffering: event.data?.is_buffering,
            chat_message: event.data?.chat_message
          }
        };
        
        console.log('UserLogs: creating log entry:', logEntry);
        newLogEntries.push(logEntry);
      }
    });
    
    if (newLogEntries.length > 0) {
      setLogs(prev => {
        // merge new entries with existing ones, avoiding duplicates
        const combined = [...newLogEntries, ...prev];
        
        // create a map for efficient duplicate detection using a composite key
        const seenKeys = new Set<string>();
        const unique = combined.filter(entry => {
          // create a unique key combining timestamp, username, action, and relevant data
          const key = `${entry.timestamp}-${entry.username}-${entry.action}-${entry.data?.current_time || 0}-${entry.data?.chat_message || ''}`;
          
          if (seenKeys.has(key)) {
            return false; // duplicate
          }
          seenKeys.add(key);
          return true; // unique
        });
        
        // sort by timestamp (newest first) and keep only last 100 entries
        return unique
          .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
          .slice(0, 100);
      });
    }
  }, [syncEvents, isAdmin]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString([], { 
      hour: '2-digit', 
      minute: '2-digit',
      second: '2-digit'
    });
  };

  const formatAction = (action: string, data?: UserLogEntry['data']) => {
    switch (action) {
      case 'play':
        return `played the video${data?.current_time ? ` at ${Math.floor(data.current_time)}s` : ''}`;
      case 'pause':
        return `paused the video${data?.current_time ? ` at ${Math.floor(data.current_time)}s` : ''}`;
      case 'seek':
        return `seeked to ${Math.floor(data?.current_time || 0)}s`;
      case 'join':
        return 'joined the room';
      case 'leave':
        return 'left the room';
      case 'buffering':
        return 'is buffering';
      case 'ready':
        return 'finished buffering';
      case 'chat':
        return `sent: "${data?.chat_message || ''}"`;
      default:
        return action;
    }
  };

  // don't render if not admin
  if (!isAdmin || !isVisible) {
    return null;
  }

  return (
    <div className="user-logs-container">
      <div className="user-logs-header">
        <span>User Activity</span>
        <div className="logs-count">{logs.length} events (received {syncEvents?.length || 0} sync events)</div>
      </div>
      
      <div className="logs-container">
        {logs.length === 0 ? (
          <div className="no-logs">No user activity yet</div>
        ) : (
          <div className="logs-list">
            {logs.map((log, index) => (
              <div key={`${log.id}-${log.timestamp}-${index}`} className="log-entry">
                <div className="log-time">{formatTime(log.timestamp)}</div>
                <div className="log-content">
                  <div className="log-user">{log.username}</div>
                  <div className="log-action">
                    {formatAction(log.action, log.data)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
