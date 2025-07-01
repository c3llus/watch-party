import React, { useState, useEffect, useRef } from 'react';
import type { ChatMessage } from '../../types/chat';
import './Chat.css';

interface ChatProps {
  messages: ChatMessage[];
  onSendMessage: (message: string) => void;
  isConnected: boolean;
  currentUsername: string;
}

export function Chat({ messages, onSendMessage, isConnected, currentUsername }: ChatProps) {
  const [inputValue, setInputValue] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (inputValue.trim() && isConnected) {
      onSendMessage(inputValue.trim());
      setInputValue('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString([], { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  return (
    <div className="chat-container">
      <div className="chat-header">
        <span>Live Chat</span>
        <div className={`connection-status ${isConnected ? 'connected' : 'disconnected'}`}>
          {isConnected ? '● Connected' : '● Disconnected'}
        </div>
      </div>
      
      <div className="messages-container">
        {messages.map((message) => {
          // handle guest username comparison - check both exact match and with " (Guest)" suffix
          const isOwnMessage = message.username === currentUsername || 
                               message.username === `${currentUsername} (Guest)`;
          return (
            <div key={message.id} className={`message-item ${isOwnMessage ? 'own-message' : ''}`}>
              <div className="message-meta">
                {!isOwnMessage && `${message.username} • `}
                {formatTime(message.timestamp)}
              </div>
              <div className={`message-bubble ${isOwnMessage ? 'own-bubble' : ''}`}>
                {message.message}
              </div>
            </div>
          );
        })}
        <div ref={messagesEndRef} />
      </div>

      <div className="input-container">
        <input
          type="text"
          className="message-input"
          placeholder={isConnected ? "type a message..." : "connecting..."}
          value={inputValue}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          disabled={!isConnected}
          maxLength={500}
        />
        <button 
          className="send-button"
          onClick={handleSend}
          disabled={!isConnected || !inputValue.trim()}
        >
          Send
        </button>
      </div>
    </div>
  );
}
