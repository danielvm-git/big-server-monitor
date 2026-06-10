import { createContext } from 'react';

export interface ToastState {
  show: (message: string, duration?: number) => void;
}

export const ToastContext = createContext<ToastState>({ show: () => {} });
