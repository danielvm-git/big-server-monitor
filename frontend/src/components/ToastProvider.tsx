import { useState, useCallback, useRef, useEffect, type ReactNode } from 'react';
import { ToastContext } from '../context/toastState';
import '../styles/animations.css';

interface ToastItem {
  id: number;
  message: string;
  leaving: boolean;
}

let nextId = 0;

export default function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const timers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  const removeToast = useCallback((id: number) => {
    timers.current.delete(id);
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  const show = useCallback((message: string, duration = 3000) => {
    const id = nextId++;
    setToasts(prev => [...prev, { id, message, leaving: false }]);
    timers.current.set(id, setTimeout(() => {
      setToasts(prev => prev.map(t => t.id === id ? { ...t, leaving: true } : t));
      setTimeout(() => removeToast(id), 200);
    }, duration));
  }, [removeToast]);

  useEffect(() => {
    return () => {
      timers.current.forEach(t => clearTimeout(t));
      timers.current.clear();
    };
  }, []);

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      <div role="status" aria-live="polite" style={{
        position: 'fixed',
        bottom: 20,
        left: '50%',
        transform: 'translateX(-50%)',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        zIndex: 9999,
      }}>
        {toasts.map(t => (
          <div
            key={t.id}
            className={t.leaving ? 'pk-fade-out' : 'pk-fade-in'}
            style={{
              background: 'var(--pop-bg)',
              color: 'var(--text)',
              border: `1px solid var(--pop-border)`,
              borderRadius: 'var(--radius-md)',
              padding: '10px 20px',
              fontSize: 13,
              fontFamily: 'var(--font)',
              boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
              whiteSpace: 'nowrap',
            }}
          >
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
