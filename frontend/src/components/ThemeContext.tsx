import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

export type ThemeId = 'xai' | 'claude' | 'sketch';

interface ThemeMeta {
  id: ThemeId;
  name: string;
  nameZh: string;
  description: string;
}

export const THEMES: ThemeMeta[] = [
  { id: 'xai', name: 'xAI', nameZh: 'xAI', description: 'Dark monospace brutalism' },
  { id: 'claude', name: 'Claude', nameZh: 'Claude', description: 'Warm cream light' },
  { id: 'sketch', name: 'Sketch', nameZh: 'Sketch 線稿', description: 'Wireframe on graph paper' },
];

interface ThemeContextValue {
  theme: ThemeId;
  setTheme: (id: ThemeId) => void;
  themes: ThemeMeta[];
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'xai',
  setTheme: () => {},
  themes: THEMES,
});

const STORAGE_KEY = 'wincmp-theme';

function getInitialTheme(): ThemeId {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && ['xai', 'claude', 'sketch'].includes(stored)) {
      return stored as ThemeId;
    }
  } catch {}
  return 'xai';
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<ThemeId>(getInitialTheme);

  const setTheme = useCallback((id: ThemeId) => {
    setThemeState(id);
    try {
      localStorage.setItem(STORAGE_KEY, id);
    } catch {}
    document.documentElement.setAttribute('data-theme', id);
  }, []);

  // Apply theme on mount
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, themes: THEMES }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}
