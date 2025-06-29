/// <reference types="vite/client" />

// HLS.js global type definitions
declare global {
  interface Window {
    Hls: {
      new (config?: HlsConfig): HlsInstance;
      isSupported(): boolean;
      Events: {
        ERROR: string;
        MANIFEST_PARSED: string;
        LEVEL_LOADED: string;
        FRAG_LOADED: string;
      };
    };
  }
  
  interface HlsConfig {
    xhrSetup?: (xhr: XMLHttpRequest, url: string) => void;
    fetchSetup?: (context: LoadContext, initParams: RequestInit) => void;
  }
  
  interface HlsInstance {
    loadSource(url: string): void;
    attachMedia(element: HTMLVideoElement): void;
    destroy(): void;
    on(event: string, callback: (event: string, data: HlsErrorData) => void): void;
  }
  
  interface LoadContext {
    url: string;
    type: string;
  }
  
  interface HlsErrorData {
    fatal: boolean;
    details: string;
    type: string;
  }
}

export {};
